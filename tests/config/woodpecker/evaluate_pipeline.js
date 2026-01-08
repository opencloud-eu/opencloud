const fs = require("fs");

const CI_REPO_NAME = process.env.CI_REPO_NAME;
const CI_COMMIT_SHA = process.env.CI_COMMIT_SHA;
const CI_WORKFLOW_NAME = process.env.CI_WORKFLOW_NAME;
const CI_PIPELINE_EVENT = process.env.CI_PIPELINE_EVENT;

const opencloudBuildWorkflow = "build-opencloud-for-testing";
const webCacheWorkflows = ["cache-web", "cache-web-pnpm", "cache-browsers"];

const INFO_URL = `https://s3.ci.opencloud.eu/public/${CI_REPO_NAME}/pipelines/${CI_COMMIT_SHA}-${CI_PIPELINE_EVENT}/pipeline_info.json`;

function getWorkflowNames(workflows) {
  const allWorkflows = [];
  for (const workflow of workflows) {
    allWorkflows.push(workflow.name);
  }
  return allWorkflows;
}

function getFailedWorkflows(workflows) {
  const failedWorkflows = [];
  for (const workflow of workflows) {
    if (workflow.state !== "success") {
      failedWorkflows.push(workflow.name);
    }
  }
  return failedWorkflows;
}

function hasFailingTestWorkflow(failedWorkflows) {
  for (const workflowName of failedWorkflows) {
    if (workflowName.startsWith("test-")) {
      return true;
    }
  }
  return false;
}

function hasFailingE2eTestWorkflow(failedWorkflows) {
  for (const workflowName of failedWorkflows) {
    if (workflowName.startsWith("test-e2e-")) {
      return true;
    }
  }
  return false;
}

async function main() {
  const infoResponse = await fetch(INFO_URL);
  if (infoResponse.status === 404) {
    console.log("[INFO] No matching previous pipeline found. Continue...");
    process.exit(0);
  } else if (!infoResponse.ok) {
    console.error(
      "[ERROR] Failed to fetch previous pipeline info:" +
        `\n  URL: ${INFO_URL}\n  Status: ${infoResponse.status}`
    );
    process.exit(1);
  }
  const info = await infoResponse.json();
  console.log(info);

  if (info.status === "success") {
    console.log(
      "[INFO] All workflows passed in previous pipeline. Full restart. Continue..."
    );
    process.exit(0);
  }

  const allWorkflows = getWorkflowNames(info.workflows);
  const failedWorkflows = getFailedWorkflows(info.workflows);

  // NOTE: implement for test pipelines only for now
  // // run the build workflow if any test workflow has failed
  // if (
  //   CI_WORKFLOW_NAME === opencloudBuildWorkflow &&
  //   hasFailingTestWorkflow(failedWorkflows)
  // ) {
  //   process.exit(0);
  // }

  // // run the web cache workflows if any e2e test workflow has failed
  // if (
  //   webCacheWorkflows.includes(CI_WORKFLOW_NAME) &&
  //   hasFailingE2eTestWorkflow(failedWorkflows)
  // ) {
  //   process.exit(0);
  // }

  if (!allWorkflows.includes(CI_WORKFLOW_NAME)) {
    process.exit(0);
  }
  if (!failedWorkflows.includes(CI_WORKFLOW_NAME)) {
    console.log("[INFO] Workflow passed in previous pipeline. Skip...");
    fs.appendFileSync(".woodpecker.env", "SKIP_WORKFLOW=true\n");
    process.exit(0);
  }
  console.log("[INFO] Restarting previously failed workflow. Continue...");
}

main();
