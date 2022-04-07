// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package io.gitpod.jetbrains.remote

import com.intellij.openapi.application.runInEdt
import com.intellij.openapi.client.ClientProjectSession
import com.jetbrains.rdserver.terminal.BackendTerminalManager
import io.gitpod.jetbrains.remote.GitpodTerminalService.Companion.TITLE_KEY
import org.jetbrains.plugins.terminal.ShellTerminalWidget
import org.jetbrains.plugins.terminal.TerminalView

@Suppress("UnstableApiUsage")
class GitpodTerminalClientService(session: ClientProjectSession) {
    init {
        runInEdt {
            val project = session.project
            val terminalView = TerminalView.getInstance(project)
            val backendTerminalManager = BackendTerminalManager.getInstance(project)
            for (widget in terminalView.widgets) {
                val widgetContent = terminalView.toolWindow.contentManager.getContent(widget)
                val terminalTitle = widgetContent.getUserData(TITLE_KEY)
                if (terminalTitle != null) {
                    backendTerminalManager.stopSharingTerminal(widget as ShellTerminalWidget)
                    backendTerminalManager.shareTerminal(widget, randomId())
                    // The following deprecated method needs to be used, otherwise not all terminals appear
                    // when the Thin Client connects.
                    // val terminalRunner = TerminalView.getRunnerByContent(widgetContent)
                    // @Suppress("DEPRECATION") terminalRunner?.openSessionInDirectory(widget, "")
                }
            }
            backendTerminalManager.connectClientToAllSharedTerminals(session.clientId)
        }
    }

    private fun randomId() = List(16) {
        (('a'..'z') + ('A'..'Z') + ('0'..'9')).random()
    }.joinToString("")
}