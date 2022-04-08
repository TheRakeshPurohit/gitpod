/**
 * Copyright (c) 2021 Gitpod GmbH. All rights reserved.
 * Licensed under the GNU Affero General Public License (AGPL).
 * See License-AGPL.txt in the project root for license information.
 */

import { AuthProviderInfo } from "@gitpod/gitpod-protocol";
import * as GitpodCookie from "@gitpod/gitpod-protocol/lib/util/gitpod-cookie";
import { useContext, useEffect, useState } from "react";
import { UserContext } from "./user-context";
import { TeamsContext } from "./teams/teams-context";
import { getGitpodService } from "./service/service";
import { iconForAuthProvider, openAuthorizeWindow, simplifyProviderName, getSafeURLRedirect } from "./provider-utils";
import gitpod from "./images/gitpod.svg";
import gitpodDark from "./images/gitpod-dark.svg";
import gitpodIcon from "./icons/gitpod.svg";
import automate from "./images/welcome/automate.svg";
import code from "./images/welcome/code.svg";
import collaborate from "./images/welcome/collaborate.svg";
import customize from "./images/welcome/customize.svg";
import fresh from "./images/welcome/fresh.svg";
import prebuild from "./images/welcome/prebuild.svg";
import exclamation from "./images/exclamation.svg";
import { getURLHash } from "./App";
import Modal from "./components/Modal";

function Item(props: { icon: string; iconSize?: string; text: string }) {
    const iconSize = props.iconSize || 28;
    return (
        <div className="flex-col items-center w-1/3 px-3">
            <img src={props.icon} className={`w-${iconSize} m-auto h-24`} />
            <div className="text-gray-400 text-sm w-36 h-20 text-center">{props.text}</div>
        </div>
    );
}

export function markLoggedIn() {
    document.cookie = GitpodCookie.generateCookie(window.location.hostname);
}

export function hasLoggedInBefore() {
    return GitpodCookie.isPresent(document.cookie);
}

export function hasVisitedMarketingWebsiteBefore() {
    return document.cookie.match("gitpod-marketing-website-visited=true");
}

export function Login() {
    const { setUser } = useContext(UserContext);
    const { setTeams } = useContext(TeamsContext);

    const urlHash = getURLHash();
    let hostFromContext: string | undefined;
    let repoPathname: string | undefined;

    try {
        if (urlHash.length > 0) {
            const url = new URL(urlHash);
            hostFromContext = url.host;
            repoPathname = url.pathname;
        }
    } catch (error) {
        // Hash is not a valid URL
    }

    const [authProviders, setAuthProviders] = useState<AuthProviderInfo[]>([]);
    const [errorMessage, setErrorMessage] = useState<string | undefined>(undefined);
    const [providerFromContext, setProviderFromContext] = useState<AuthProviderInfo>();
    const [showCookieBanner, setShowCookieBanner] = useState<boolean>(true);
    const [showCookieSettings, setShowCookieSettings] = useState<boolean>(false);

    const showWelcome = !hasLoggedInBefore() && !hasVisitedMarketingWebsiteBefore() && !urlHash.startsWith("https://");

    useEffect(() => {
        (async () => {
            setAuthProviders(await getGitpodService().server.getAuthProviders());
        })();
    }, []);

    useEffect(() => {
        if (hostFromContext && authProviders) {
            const providerFromContext = authProviders.find((provider) => provider.host === hostFromContext);
            setProviderFromContext(providerFromContext);
        }
    }, [authProviders]);

    const authorizeSuccessful = async (payload?: string) => {
        updateUser().catch(console.error);

        // Check for a valid returnTo in payload
        const safeReturnTo = getSafeURLRedirect(payload);
        if (safeReturnTo) {
            // ... and if it is, redirect to it
            window.location.replace(safeReturnTo);
        }
    };

    const updateUser = async () => {
        await getGitpodService().reconnect();
        const [user, teams] = await Promise.all([
            getGitpodService().server.getLoggedInUser(),
            getGitpodService().server.getTeams(),
        ]);
        setUser(user);
        setTeams(teams);
        markLoggedIn();
    };

    const openLogin = async (host: string) => {
        setErrorMessage(undefined);

        try {
            await openAuthorizeWindow({
                login: true,
                host,
                onSuccess: authorizeSuccessful,
                onError: (payload) => {
                    let errorMessage: string;
                    if (typeof payload === "string") {
                        errorMessage = payload;
                    } else {
                        errorMessage = payload.description ? payload.description : `Error: ${payload.error}`;
                        if (payload.error === "email_taken") {
                            errorMessage = `Email address already used in another account. Please log in with ${
                                (payload as any).host
                            }.`;
                        }
                    }
                    setErrorMessage(errorMessage);
                },
            });
        } catch (error) {
            console.log(error);
        }
    };

    const closeCookieBanner = (response: string) => {
        setShowCookieBanner(false);
    };

    const openCookieSettings = () => {
        console.log(showCookieSettings);
        setShowCookieSettings(true);
    };

    return (
        <div id="login-container" className="z-50 flex w-screen h-screen">
            {showCookieSettings && (
                <Modal visible={showCookieSettings} onClose={() => setShowCookieSettings(false)}>
                    <h3>Cookie settings</h3>
                    <h4>Strictly necessary cookies</h4>
                    <p>
                        These are cookies that are required for the operation of our Site and under our terms with you.
                        They include, for example, cookies that enable you to log into secure areas of our Site or (on
                        other sites) use a shopping cart or make use of e-billing services.
                    </p>
                    <label className="relative inline-block	w-12 h-7 rounded-3xl" style={{ backgroundColor: "grey" }}>
                        <input className="opacity-0" type="checkbox"></input>
                        <span
                            className="absolute w-full h-full cursor-pointer"
                            style={{ transition: "backgroundColor .4s ease-out" }}
                        ></span>
                    </label>
                    <h4>Analytical / Performance cookies</h4>
                    <p>
                        These allow us to recognise and count the number of visitors and to see how visitors move around
                        our site when they are using it. This helps us improve the way our Website works, for example,
                        by ensuring that users are finding what they are looking for easily.
                    </p>
                    <h4>Targeting/Advertising cookies</h4>
                    <p>
                        These cookies record your visit to our Website, the pages you have visited and the links you
                        have followed. We will use this information subject to your choices and preferences to make our
                        Website and the advertising displayed on it more relevant to your interests. We may also share
                        this information with third parties for this purpose.
                    </p>
                </Modal>
            )}
            {showWelcome ? (
                <div id="feature-section" className="flex-grow bg-gray-100 dark:bg-gray-800 w-1/2 hidden lg:block">
                    <div id="feature-section-column" className="flex max-w-xl h-full mx-auto pt-6">
                        <div className="flex flex-col px-8 my-auto ml-auto">
                            <div className="mb-12">
                                <img src={gitpod} className="h-8 block dark:hidden" alt="Gitpod light theme logo" />
                                <img src={gitpodDark} className="h-8 hidden dark:block" alt="Gitpod dark theme logo" />
                            </div>
                            <div className="mb-10">
                                <h1 className="text-5xl mb-3">Welcome to Gitpod</h1>
                                <div className="text-gray-400 text-lg">
                                    Spin up fresh, automated dev environments for each task in the cloud, in seconds.
                                </div>
                            </div>
                            <div className="flex mb-10">
                                <Item icon={code} iconSize="16" text="Always Ready&#x2011;To&#x2011;Code" />
                                <Item icon={customize} text="Personalize your Workspace" />
                                <Item icon={automate} text="Automate Your Development Setup" />
                            </div>
                            <div className="flex">
                                <Item icon={prebuild} text="Continuously Prebuild Your Project" />
                                <Item icon={collaborate} text="Collaborate With Your Team" />
                                <Item icon={fresh} text="Fresh Workspace For Each New Task" />
                            </div>
                        </div>
                    </div>
                </div>
            ) : null}
            <div id="login-section" className={"flex-grow w-full" + (showWelcome ? " lg:w-1/2" : "")}>
                <div
                    id="login-section-column"
                    className={"flex-grow max-w-2xl flex flex-col h-100 mx-auto" + (showWelcome ? " lg:my-0" : "")}
                >
                    <div className="flex-grow h-100 flex flex-row items-center justify-center">
                        <div className="rounded-xl px-10 py-10 mx-auto">
                            <div className="mx-auto pb-8">
                                <img
                                    src={providerFromContext ? gitpod : gitpodIcon}
                                    className="h-14 mx-auto block dark:hidden"
                                    alt="Gitpod's logo"
                                />
                                <img
                                    src={providerFromContext ? gitpodDark : gitpodIcon}
                                    className="h-14 hidden mx-auto dark:block"
                                    alt="Gitpod dark theme logo"
                                />
                            </div>

                            <div className="mx-auto text-center pb-8 space-y-2">
                                {providerFromContext ? (
                                    <>
                                        <h2 className="text-xl text-black dark:text-gray-50 font-semibold">
                                            Open a cloud-based developer environment
                                        </h2>
                                        <h2 className="text-xl">for the repository {repoPathname?.slice(1)}</h2>
                                    </>
                                ) : (
                                    <>
                                        <h1 className="text-3xl">Log in{showWelcome ? "" : " to Gitpod"}</h1>
                                        <h2 className="uppercase text-sm text-gray-400">ALWAYS READY-TO-CODE</h2>
                                    </>
                                )}
                            </div>

                            <div className="flex flex-col space-y-3 items-center">
                                {providerFromContext ? (
                                    <button
                                        key={"button" + providerFromContext.host}
                                        className="btn-login flex-none w-56 h-10 p-0 inline-flex"
                                        onClick={() => openLogin(providerFromContext.host)}
                                    >
                                        {iconForAuthProvider(providerFromContext.authProviderType)}
                                        <span className="pt-2 pb-2 mr-3 text-sm my-auto font-medium truncate overflow-ellipsis">
                                            Continue with {simplifyProviderName(providerFromContext.host)}
                                        </span>
                                    </button>
                                ) : (
                                    authProviders.map((ap) => (
                                        <button
                                            key={"button" + ap.host}
                                            className="btn-login flex-none w-56 h-10 p-0 inline-flex"
                                            onClick={() => openLogin(ap.host)}
                                        >
                                            {iconForAuthProvider(ap.authProviderType)}
                                            <span className="pt-2 pb-2 mr-3 text-sm my-auto font-medium truncate overflow-ellipsis">
                                                Continue with {simplifyProviderName(ap.host)}
                                            </span>
                                        </button>
                                    ))
                                )}
                            </div>

                            {errorMessage && (
                                <div className="mt-16 flex space-x-2 py-6 px-6 w-96 justify-between bg-gitpod-kumquat-light rounded-xl">
                                    <div className="pr-3 self-center w-6">
                                        <img src={exclamation} />
                                    </div>
                                    <div className="flex-1 flex flex-col">
                                        <p className="text-gitpod-red text-sm">{errorMessage}</p>
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
                {showCookieBanner && (
                    <div className="flex justify-between items-center mx-auto h-12 px-4 py-2 text-center text-xs text-gray-600 bg-sand-dark w-full bottom-0 left-0 fixed">
                        <p className="text-gray-600">
                            The website uses cookies to enhance the user experience. Read our{" "}
                            <a
                                className="gp-link hover:text-gray-600"
                                target="gitpod-privacy"
                                href="https://www.gitpod.io/privacy/"
                            >
                                privacy policy{" "}
                            </a>
                            for more info.
                        </p>
                        <div className="flex gap-1">
                            <button
                                className="py-3 bg-sand-dark underline text-xs text-gray-400 hover:text-gray-600"
                                onClick={() => openCookieSettings()}
                            >
                                Modify settings
                            </button>
                            <button
                                className="bg-off-white rounded-lg hover:bg-white text-xs text-gray-600"
                                onClick={() => closeCookieBanner("accepted")}
                            >
                                Accept Cookies
                            </button>
                            <button
                                className="bg-off-white rounded-lg hover:bg-white text-xs text-gray-600"
                                onClick={() => closeCookieBanner("rejected")}
                            >
                                Reject All
                            </button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}
