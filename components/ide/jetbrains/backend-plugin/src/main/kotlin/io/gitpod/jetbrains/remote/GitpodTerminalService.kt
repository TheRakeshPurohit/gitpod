// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package io.gitpod.jetbrains.remote

import com.google.protobuf.ByteString
import com.intellij.openapi.application.runInEdt
import com.intellij.openapi.diagnostic.thisLogger
import com.intellij.openapi.project.Project
import com.intellij.openapi.util.Key
import com.intellij.openapi.wm.ToolWindowManager
import com.intellij.openapi.wm.ex.ToolWindowManagerListener
import com.jediterm.terminal.TtyConnector
import com.jetbrains.rdserver.terminal.BackendTerminalManager
import com.jetbrains.rdserver.terminal.BackendTtyConnector
import io.gitpod.supervisor.api.Status.*
import io.gitpod.supervisor.api.StatusServiceGrpc
import io.gitpod.supervisor.api.TerminalOuterClass
import io.gitpod.supervisor.api.TerminalServiceGrpc
import io.grpc.StatusRuntimeException
import io.grpc.stub.StreamObserver
import kotlinx.coroutines.*
import org.jetbrains.plugins.terminal.ShellTerminalWidget
import org.jetbrains.plugins.terminal.TerminalTabState
import org.jetbrains.plugins.terminal.TerminalToolWindowFactory
import org.jetbrains.plugins.terminal.TerminalView
import org.jetbrains.plugins.terminal.cloud.CloudTerminalProcess
import org.jetbrains.plugins.terminal.cloud.CloudTerminalRunner
import java.io.ByteArrayOutputStream
import java.io.PipedInputStream
import java.io.PipedOutputStream

@Suppress("UnstableApiUsage")
class GitpodTerminalService(private val project: Project) {
    companion object {
        val TITLE_KEY = Key.create<String>("TITLE_KEY")
    }

    private val terminalView = TerminalView.getInstance(project)
    private val terminalServiceStub = TerminalServiceGrpc.newStub(GitpodManager.supervisorChannel)
    private val statusServiceStub = StatusServiceGrpc.newStub(GitpodManager.supervisorChannel)
    private val backendTerminalManager = BackendTerminalManager.getInstance(project)

    init {
        afterTerminalToolWindowGetsRegistered {
            withSupervisorTasksList { tasksList ->
                withSupervisorTerminalsList { terminalsList ->
                    runInEdt {
                        for (terminalWidget in terminalView.widgets) {
                            val terminalContent = terminalView.toolWindow.contentManager.getContent(terminalWidget)
                            val terminalTitle = terminalContent.getUserData(TITLE_KEY)
                            if (terminalTitle != null) {
                                debug("Closing terminal $terminalTitle before opening it again.")
                                terminalWidget.close()
                            }
                        }

                        if (tasksList.isEmpty() || terminalsList.isEmpty()) {
                            backendTerminalManager.createNewSharedTerminal(
                                    "GitpodTerminal",
                                    "Terminal"
                            )
                        } else {
                            val aliasToTerminalMap:
                                    MutableMap<String, TerminalOuterClass.Terminal> =
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
        }
    }

    private fun afterTerminalToolWindowGetsRegistered(action: () -> Unit) {
        debug("Waiting for TerminalToolWindow to be registered...")
        val toolWindowManagerListener =
                object : ToolWindowManagerListener {
                    override fun toolWindowsRegistered(
                            ids: MutableList<String>,
                            toolWindowManager: ToolWindowManager
                    ) {
                        if (ids.contains(TerminalToolWindowFactory.TOOL_WINDOW_ID)) {
                            debug("TerminalToolWindow got registered!")
                            action()
                        }
                    }
                }

        project.messageBus
                .connect()
                .subscribe(ToolWindowManagerListener.TOPIC, toolWindowManagerListener)
    }

    private fun withSupervisorTasksList(action: (tasksList: List<TaskStatus>) -> Unit) {
        val taskStatusRequest = TasksStatusRequest.newBuilder().setObserve(true).build()

        val taskStatusResponseObserver =
                object : StreamObserver<TasksStatusResponse> {
                    override fun onNext(response: TasksStatusResponse) {
                        debug("Received task list: ${response.tasksList}")

                        var hasOpenedAllTasks = true

                        response.tasksList.forEach { task ->
                            if (task.state === TaskState.opening) {
                                hasOpenedAllTasks = false
                            }
                        }

                        if (hasOpenedAllTasks) {
                            this.onCompleted()
                            action(response.tasksList)
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
                    }
                }

        statusServiceStub.tasksStatus(taskStatusRequest, taskStatusResponseObserver)
    }

    private fun withSupervisorTerminalsList(
            action: (terminalsList: List<TerminalOuterClass.Terminal>) -> Unit
    ) {
        val listTerminalsRequest = TerminalOuterClass.ListTerminalsRequest.newBuilder().build()

        val listTerminalsResponseObserver =
                object : StreamObserver<TerminalOuterClass.ListTerminalsResponse> {
                    override fun onNext(response: TerminalOuterClass.ListTerminalsResponse) {
                        debug("Got a list of Supervisor terminals: ${response.terminalsList}")
                        action(response.terminalsList)
                    }

                    override fun onError(throwable: Throwable) {
                        thisLogger()
                                .error(
                                        "Got an error while getting the list of Supervisor terminals.",
                                        throwable
                                )
                    }

                    override fun onCompleted() {
                        debug("Successfully got the list of Supervisor terminals.")
                    }
                }

        terminalServiceStub.list(listTerminalsRequest, listTerminalsResponseObserver)
    }

    private fun createSharedTerminal(supervisorTerminal: TerminalOuterClass.Terminal) {
        debug("Creating shared terminal '${supervisorTerminal.title}' on Backend IDE")
        val terminalInputReader = PipedInputStream()
        val terminalInputWriter = PipedOutputStream(terminalInputReader)
        val terminalOutputReader = PipedInputStream()
        val terminalOutputWriter = PipedOutputStream(terminalOutputReader)
        val terminalProcess = CloudTerminalProcess(terminalInputWriter, terminalOutputReader)
        val terminalRunner =
                object : CloudTerminalRunner(project, supervisorTerminal.alias, terminalProcess) {
                    override fun createTtyConnector(process: CloudTerminalProcess): TtyConnector {
                        return BackendTtyConnector(project, super.createTtyConnector(process))
                    }
                }

        val terminalRunnerId = terminalRunner.toString()

        terminalView.createNewSession(
                terminalRunner,
                TerminalTabState().also { it.myTabName = terminalRunnerId }
        )

        val shellTerminalWidget =
                terminalView.widgets.find { widget ->
                    terminalView.toolWindow.contentManager.getContent(widget).tabName ==
                    terminalRunnerId
                } as ShellTerminalWidget

        backendTerminalManager.shareTerminal(shellTerminalWidget, terminalRunnerId)

        val terminalContent = terminalView.toolWindow.contentManager.getContent(shellTerminalWidget)

        terminalContent.putUserData(TITLE_KEY, supervisorTerminal.title)

        connectSupervisorStream(
                shellTerminalWidget,
                supervisorTerminal,
                terminalOutputWriter,
                terminalInputReader
        )
    }

    private fun connectSupervisorStream(
            shellTerminalWidget: ShellTerminalWidget,
            supervisorTerminal: TerminalOuterClass.Terminal,
            terminalOutputWriter: PipedOutputStream,
            terminalInputReader: PipedInputStream
    ) {
        val dataReceivedFromSupervisor = ByteArrayOutputStream()

        val listenTerminalRequest =
                TerminalOuterClass.ListenTerminalRequest.newBuilder()
                        .setAlias(supervisorTerminal.alias)
                        .build()

        val listenTerminalResponseObserver =
                object : StreamObserver<TerminalOuterClass.ListenTerminalResponse> {
                    override fun onNext(response: TerminalOuterClass.ListenTerminalResponse) {
                        when {
                            response.hasTitle() -> {
                                debug("Received terminal title: ${response.title}")
                                val terminalContent = terminalView.toolWindow.contentManager.getContent(shellTerminalWidget)
                                terminalContent.putUserData(TITLE_KEY, response.title)
                            }
                            response.hasData() -> {
                                debug("Printing a text on '${supervisorTerminal.title}' terminal.")
                                dataReceivedFromSupervisor.write(response.data.toByteArray())
                            }
                            response.hasExitCode() -> {
                                debug(
                                        "Closing '${supervisorTerminal.title}' terminal (Exit Code: ${response.exitCode}."
                                )
                                shellTerminalWidget.close()
                            }
                        }
                    }

                    override fun onCompleted() {
                        debug("'${supervisorTerminal.title}' terminal finished reading stream.")
                    }

                    override fun onError(throwable: Throwable) {
                        if (containsTerminatedStatus(throwable)) return

                        thisLogger()
                                .error(
                                        "Got an error while listening to '${supervisorTerminal.title}' terminal.",
                                        throwable
                                )
                    }
                }

        terminalServiceStub.listen(listenTerminalRequest, listenTerminalResponseObserver)

        /** Controls the writing flow, ensuring we send only one write request at a time. */
        var isWaitingForWritingResponse = false

        val writeTerminalResponseObserver =
                object : StreamObserver<TerminalOuterClass.WriteTerminalResponse> {
                    override fun onNext(response: TerminalOuterClass.WriteTerminalResponse) {
                        debug(
                                "${response.bytesWritten} bytes written on '${supervisorTerminal.title}' terminal."
                        )
                        isWaitingForWritingResponse = false
                    }

                    override fun onError(throwable: Throwable) {
                        isWaitingForWritingResponse = false

                        if (containsTerminatedStatus(throwable)) return

                        thisLogger()
                                .error(
                                        "Got an error while writing to '${supervisorTerminal.title}' terminal.",
                                        throwable
                                )
                    }

                    override fun onCompleted() {
                        debug("'${supervisorTerminal.title}' terminal finished writing stream.")
                        isWaitingForWritingResponse = false
                    }
                }

        val writeTerminalRequestBuilder =
                TerminalOuterClass.WriteTerminalRequest.newBuilder()
                        .setAlias(supervisorTerminal.alias)

        @Suppress("EXPERIMENTAL_IS_NOT_ENABLED")
        @OptIn(DelicateCoroutinesApi::class)
        val watchTerminalInputJob =
                GlobalScope.launch {
                    withContext(Dispatchers.IO) {
                        while (isActive) {
                            if (dataReceivedFromSupervisor.size() > 0) {
                                val bytes = dataReceivedFromSupervisor.toByteArray()
                                dataReceivedFromSupervisor.reset()
                                terminalOutputWriter.write(bytes)
                                terminalOutputWriter.flush()
                            }

                            val bytesAvailableOnTerminalInput = terminalInputReader.available()

                            if (!isWaitingForWritingResponse && bytesAvailableOnTerminalInput > 0) {
                                isWaitingForWritingResponse = true

                                val bytesFromTerminalInput =
                                        ByteArray(bytesAvailableOnTerminalInput)

                                terminalInputReader.read(
                                        bytesFromTerminalInput,
                                        0,
                                        bytesAvailableOnTerminalInput
                                )

                                val supervisorTerminalStdin =
                                        ByteString.copyFrom(bytesFromTerminalInput)
                                val writeTerminalRequest =
                                        writeTerminalRequestBuilder
                                                .setStdin(supervisorTerminalStdin)
                                                .build()

                                terminalServiceStub.write(
                                        writeTerminalRequest,
                                        writeTerminalResponseObserver
                                )
                            }
                        }
                    }
                }

        shellTerminalWidget.addListener {
            debug("Terminal '${supervisorTerminal.title}' was closed via IDE.")
            watchTerminalInputJob.cancel()
            val shutdownTerminalRequest =
                    TerminalOuterClass.ShutdownTerminalRequest.newBuilder()
                            .setAlias(supervisorTerminal.alias)
                            .build()
            val shutdownTerminalResponseObserver =
                    object : StreamObserver<TerminalOuterClass.ShutdownTerminalResponse> {
                        override fun onNext(response: TerminalOuterClass.ShutdownTerminalResponse) =
                                Unit

                        override fun onError(throwable: Throwable) {
                            if (containsTerminatedStatus(throwable)) return

                            thisLogger()
                                    .error(
                                            "Got an error while trying to shutdown '${supervisorTerminal.title}' terminal.",
                                            throwable
                                    )
                        }

                        override fun onCompleted() {
                            debug("Successfully shutdown '${supervisorTerminal.title}' terminal.")
                        }
                    }
            terminalServiceStub.shutdown(shutdownTerminalRequest, shutdownTerminalResponseObserver)
        }
    }

    private fun containsTerminatedStatus(throwable: Throwable) =
            throwable is StatusRuntimeException &&
                    throwable.status.description == "signal: terminated"

    private fun debug(message: String) = runInEdt {
        if (System.getenv("JB_DEV").toBoolean()) thisLogger().warn(message)
    }
}
