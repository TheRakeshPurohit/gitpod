/**
 * Copyright (c) 2021 Gitpod GmbH. All rights reserved.
 * Licensed under the GNU Affero General Public License (AGPL).
 * See License-AGPL.txt in the project root for license information.
 */

import Alert from "./Alert";

export function Section(props: { title: string; children: React.ReactNode | React.ReactNode[]; darkMode?: boolean }) {
    const darkMode = props.darkMode ?? true;
    return (
        <>
            <h3 className="text-yellow-800 mt-4">{props.title}</h3>
            <div className={`grid gap-2 ${darkMode ? "grid-cols-2" : "grid-cols-1"}`}>
                <div className="space-y-2 p-4 bg-white rounded">{props.children}</div>
                {darkMode && <div className="space-y-2 p-4 dark bg-black rounded">{props.children}</div>}
            </div>
        </>
    );
}

export default function Display() {
    return (
        <div className="bg-gitpod-kumquat pb-20">
            <div className="container">
                <h1 className="text-yellow-800">Components Display</h1>
                <Section title="Alert">
                    <Alert closable={true} type="error">
                        error
                    </Alert>
                    <Alert closable={true} type="info">
                        <strong>Info:</strong> Cupidatat minim culpa voluptate incididunt ad consectetur magna fugiat
                        pariatur. Nisi aliquip nostrud sunt laboris laboris incididunt Lorem officia qui id. Consectetur
                        qui nulla deserunt excepteur consectetur deserunt qui qui nisi sint eiusmod ad fugiat tempor.
                    </Alert>
                    <Alert closable={true} type="warning">
                        warning
                    </Alert>
                    <Alert closable={true} type="message">
                        message
                    </Alert>
                    <Alert showIcon={false} type="warning">
                        warning without icon shown and closable. Ea consequat in sint in deserunt adipisicing commodo
                        mollit aliquip est nostrud. Minim exercitation pariatur non cupidatat consectetur pariatur.
                        Velit deserunt minim id laboris magna fugiat proident laborum velit laboris amet tempor
                        consequat in. Amet voluptate qui commodo adipisicing cillum sit duis dolore fugiat ipsum
                        deserunt.
                    </Alert>
                    <Alert showIcon={false} closable type="info">
                        Aute irure esse officia fugiat ullamco. Amet mollit ipsum esse cillum nostrud sit enim ullamco
                        pariatur eiusmod cillum. Nisi dolor in duis aute eu adipisicing irure sunt cillum in.
                    </Alert>
                </Section>
            </div>
        </div>
    );
}
