// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package io.gitpod.jetbrains.remote

import com.intellij.openapi.application.runInEdt
import com.intellij.openapi.client.ClientProjectSession
import org.jetbrains.plugins.terminal.TerminalView

@Suppress("UnstableApiUsage")
class GitpodTerminalClientService(session: ClientProjectSession) {
    init {
        runInEdt {
            val project = session.project
            val terminalView = TerminalView.getInstance(project)
            for (widget in terminalView.widgets) {
                val widgetContent = terminalView.toolWindow.contentManager.getContent(widget)
                val terminalRunner = TerminalView.getRunnerByContent(widgetContent)
                // The following deprecated method needs to be used, otherwise not all terminals appear
                // when the Thin Client connects.
                @Suppress("DEPRECATION") terminalRunner?.openSessionInDirectory(widget, "")
            }
        }
    }
}