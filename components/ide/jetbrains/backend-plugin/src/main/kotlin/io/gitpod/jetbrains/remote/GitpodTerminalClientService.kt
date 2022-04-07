// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package io.gitpod.jetbrains.remote

import com.intellij.openapi.Disposable
import com.intellij.openapi.application.runInEdt
import com.intellij.openapi.client.ClientProjectSession
import com.intellij.openapi.diagnostic.thisLogger
import com.intellij.openapi.util.Key
import com.jetbrains.rd.util.lifetime.Lifetime
import com.jetbrains.rdserver.terminal.BackendTerminalManager
import io.gitpod.supervisor.api.Status
import io.gitpod.supervisor.api.StatusServiceGrpc
import io.gitpod.supervisor.api.TerminalOuterClass
import io.gitpod.supervisor.api.TerminalServiceGrpc
import io.grpc.stub.StreamObserver
import kotlinx.coroutines.Deferred
import kotlinx.coroutines.DelicateCoroutinesApi
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.future.await
import kotlinx.coroutines.guava.asDeferred
import kotlinx.coroutines.launch
import org.jetbrains.plugins.terminal.TerminalView
import java.util.concurrent.CompletableFuture

@Suppress("EXPERIMENTAL_IS_NOT_ENABLED", "UnstableApiUsage", "OPT_IN_IS_NOT_ENABLED")
@OptIn(DelicateCoroutinesApi::class)
class GitpodTerminalClientService(session: ClientProjectSession) : Disposable {
    companion object {
        val TITLE_KEY = Key.create<String>("TITLE_KEY")
    }

    private val project = session.project
    private val lifetime = Lifetime.Eternal.createNested()
    private val terminalView = TerminalView.getInstance(project)
    private val terminalServiceFutureStub =
            TerminalServiceGrpc.newFutureStub(GitpodManager.supervisorChannel)
    private val statusServiceStub = StatusServiceGrpc.newStub(GitpodManager.supervisorChannel)
    private val backendTerminalManager = BackendTerminalManager.getInstance(project)

    override fun dispose() {
        lifetime.terminate()
    }

    init {
        GlobalScope.launch {
            val tasksList = getSupervisorTasksList().await()
            val terminalsList = getSupervisorTerminalsListAsync().await().terminalsList
            debug("Got a list of Supervisor terminals: $terminalsList")
            runInEdt {
//                for (terminalWidget in terminalView.widgets) {
//                    val terminalContent =
//                            terminalView.toolWindow.contentManager.getContent(terminalWidget)
//                    val terminalTitle = terminalContent.getUserData(TITLE_KEY)
//                    if (terminalTitle != null) {
//                        debug("Closing terminal $terminalTitle before opening it again.")
//                        terminalWidget.close()
//                    }
//                }

                if (tasksList.isEmpty() || terminalsList.isEmpty()) {
                    backendTerminalManager.createNewSharedTerminal("GitpodTerminal", "Terminal")
                } else {
                    val aliasToTerminalMap: MutableMap<String, TerminalOuterClass.Terminal> =
                            mutableMapOf()

                    for (terminal in terminalsList) {
                        val terminalAlias = terminal.alias
                        aliasToTerminalMap[terminalAlias] = terminal
                    }

                    for (task in tasksList) {
                        val terminalAlias = task.terminal
                        val terminal = aliasToTerminalMap[terminalAlias]

                        if (terminal != null) {
                            createSharedTerminal(terminal)
                        }
                    }
                }
            }
        }
    }

    private fun getSupervisorTasksList(): CompletableFuture<List<Status.TaskStatus>> {
        val completableFuture = CompletableFuture<List<Status.TaskStatus>>()
        val taskStatusRequest = Status.TasksStatusRequest.newBuilder().setObserve(true).build()
        val taskStatusResponseObserver =
                object : StreamObserver<Status.TasksStatusResponse> {
                    override fun onNext(response: Status.TasksStatusResponse) {
                        debug("Received task list: ${response.tasksList}")

                        var hasOpenedAllTasks = true

                        response.tasksList.forEach { task ->
                            if (task.state === Status.TaskState.opening) {
                                hasOpenedAllTasks = false
                            }
                        }

                        if (hasOpenedAllTasks) {
                            this.onCompleted()
                            completableFuture.complete(response.tasksList)
                        }
                    }

                    override fun onCompleted() {
                        debug("Successfully fetched tasks from Supervisor.")
                    }

                    override fun onError(throwable: Throwable) {
                        thisLogger()
                                .error(
                                        "Got an error while trying to fetch tasks from Supervisor.",
                                        throwable
                                )
                        completableFuture.completeExceptionally(throwable)
                    }
                }

        statusServiceStub.tasksStatus(taskStatusRequest, taskStatusResponseObserver)

        return completableFuture
    }

    private fun getSupervisorTerminalsListAsync(): Deferred<TerminalOuterClass.ListTerminalsResponse> {
        val listTerminalsRequest = TerminalOuterClass.ListTerminalsRequest.newBuilder().build()
        return terminalServiceFutureStub.list(listTerminalsRequest).asDeferred()
    }

    private fun createSharedTerminal(supervisorTerminal: TerminalOuterClass.Terminal) {
        debug("Creating shared terminal '${supervisorTerminal.title}' on Backend IDE")

        val shellTerminalWidget = terminalView.createLocalShellWidget(project.basePath, supervisorTerminal.title, false)

        val terminalContent = terminalView.toolWindow.contentManager.getContent(shellTerminalWidget)

        terminalContent.putUserData(TITLE_KEY, supervisorTerminal.title)

        val terminalRunner = TerminalView.getRunnerByContent(terminalContent)

        backendTerminalManager.shareTerminal(shellTerminalWidget, terminalRunner.toString())

        // The following deprecated method needs to be used, otherwise not all terminals appear
        // when the Thin Client connects.
        // @Suppress("DEPRECATION") terminalRunner?.openSessionInDirectory(shellTerminalWidget, "")

        shellTerminalWidget.executeCommand("gp tasks attach ${supervisorTerminal.alias}")
    }

    private fun debug(message: String) = runInEdt {
        if (System.getenv("JB_DEV").toBoolean()) thisLogger().warn(message)
    }
}