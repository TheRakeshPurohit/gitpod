import { Werft } from './util/werft';
import * as Tracing from './observability/tracing';
import { SpanStatusCode } from '@opentelemetry/api';
import { wipePreviewEnvironmentAndNamespace, helmInstallName, listAllPreviewNamespaces } from './util/kubectl';
import { exec } from './util/shell';
import { previewNameFromBranchName } from './util/preview';
import { CORE_DEV_KUBECONFIG_PATH } from './jobs/build/const';

// Will be set once tracing has been initialized
let werft: Werft

const previewsToDelete = [];

Tracing.initialize()
    .then(() => {
        werft = new Werft("delete-preview-environment-cron")
    })
    .then(() => deletePreviewEnvironments())
    .catch((err) => {
        werft.rootSpan.setStatus({
            code: SpanStatusCode.ERROR,
            message: err
        })
    })
    .finally(() => {
        werft.phase("Flushing telemetry", "Flushing telemetry before stopping job")
        werft.endAllSpans()
    })

async function deletePreviewEnvironments() {

    werft.phase("prep");
    try {
        const GCLOUD_SERVICE_ACCOUNT_PATH = "/mnt/secrets/gcp-sa/service-account.json";
        exec(`gcloud auth activate-service-account --key-file "${GCLOUD_SERVICE_ACCOUNT_PATH}"`);
        exec(`KUBECONFIG=${CORE_DEV_KUBECONFIG_PATH} gcloud container clusters get-credentials core-dev --zone europe-west1-b --project gitpod-core-dev`);
    } catch (err) {
        werft.fail("prep", err)
    }
    werft.done("prep")

    werft.phase("Fetching branches");
    const branches = getAllBranches();
    // During the transition from the old preview names to the new ones we have to check for the existence of both the old or new
    // preview name patterns before it is safe to delete a namespace.
    const expectedPreviewEnvironmentNamespaces = new Set(branches.flatMap(branch => [parseBranch(branch), expectedNamespaceFromBranch(branch)]));
    werft.done("Fetching branches");

    werft.phase("Fetching previews");
    let previews: string[]
    try {
        previews = listAllPreviewNamespaces(CORE_DEV_KUBECONFIG_PATH, {});
        previews.forEach(previewNs => werft.log("Fetching previews", previewNs))
    } catch (err) {
        werft.fail("Fetching previews", err)
    }
    werft.done("Fetching previews");


    werft.phase("checking activity")
    const promises: Promise<any>[] = [];
    previews.forEach(previewNS => {
        promises.push(checkDB(previewNS));
    })
    await Promise.all(promises);

    werft.log("previewToDelete", previewsToDelete)

    werft.phase("deleting previews")
    try {
        // const previewsToDelete = previews.filter(ns => !expectedPreviewEnvironmentNamespaces.has(ns))
        previewsToDelete.forEach(previewNs => werft.log("Deleting previews", previewNs))
        // Trigger namespace deletion in parallel
        // const promises = previewsToDelete.map(preview => wipePreviewEnvironmentAndNamespace(helmInstallName, preview, CORE_DEV_KUBECONFIG_PATH, { slice: `Deleting preview ${preview}` }));
        // // But wait for all of them to finish before (or one of them to fail) before we continue
        // await Promise.all(promises)
    } catch (err) {
        werft.fail("deleting previews", err)
    }
    werft.done("deleting previews")
}

async function checkDB(previewNS) {

    werft.log("checking namespace", previewNS)

    const statusNS = exec(`KUBECONFIG=${CORE_DEV_KUBECONFIG_PATH} kubectl get ns ${previewNS} -o jsonpath='{.status.phase}'`, { silent: true})

    if ( statusNS == "Active") {

        const statusDB = exec(`KUBECONFIG=${CORE_DEV_KUBECONFIG_PATH} kubectl get pods mysql-0 -n ${previewNS} -o jsonpath='{.status.phase}'`, { silent: true})

        if (statusDB == "Running" ) {

            const connectionToDb = `KUBECONFIG=${CORE_DEV_KUBECONFIG_PATH} kubectl get secret db-password -n ${previewNS} -o jsonpath='{.data.mysql-password}' | base64 -d | mysql --host=db.${previewNS}.svc.cluster.local --port=3306 --user=gitpod --database=gitpod -s -N -p`

            const latestInstanceTimeout = 24
            const latestInstance = exec(`${connectionToDb} --execute="SELECT creationTime FROM d_b_workspace_instance WHERE creationTime > DATE_SUB(NOW(), INTERVAL '${latestInstanceTimeout}' HOUR) LIMIT 1"`, { silent: true })

            const latestUserTimeout = 24
            const latestUser= exec(`${connectionToDb} --execute="SELECT creationDate FROM d_b_user WHERE creationDate > DATE_SUB(NOW(), INTERVAL '${latestUserTimeout}' HOUR) LIMIT 1"`, { silent: true })

            const heartbeatTimeout = 24
            const heartbeat= exec(`${connectionToDb} --execute="SELECT lastSeen FROM d_b_workspace_instance_user WHERE lastSeen > DATE_SUB(NOW(), INTERVAL '${heartbeatTimeout}' HOUR) LIMIT 1"`, { silent: true })

            if ( !(heartbeat.length > 4) && !(latestInstance.length > 4) && !(latestUser.length > 4) ) {
                previewsToDelete.push(previewNS)
            }
        }
    }

}

function getAllBranches(): string[] {
    return exec(`git branch -r | grep -v '\\->' | sed "s,\\x1B\\[[0-9;]*[a-zA-Z],,g" | while read remote; do echo "\${remote#origin/}"; done`).stdout.trim().split('\n');
}

function expectedNamespaceFromBranch(branch: string): string {
    const previewName = previewNameFromBranchName(branch)
    return `staging-${previewName}`
}

function parseBranch(branch: string): string {
    const prefix = 'staging-';
    const parsedBranch = branch.normalize().split("/").join("-");

    return prefix + parsedBranch;
}
