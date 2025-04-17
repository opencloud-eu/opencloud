export default {
  changeTypes: [
    {
      title: "💥 Breaking changes",
      labels: ["breaking", "Type:Breaking-Change"],
      bump: "major",
      weight: 3,
    },
    {
      title: "🔒 Security",
      labels: ["security", "Type:Security"],
      bump: "patch",
      weight: 2,
    },
    {
      title: "✨ Features",
      labels: ["feature", "Type:Feature"],
      bump: "minor",
      weight: 1,
    },
    {
      title: "📈 Enhancement",
      labels: ["enhancement", "refactor", "Type:Enhancement"],
      bump: "minor",
    },
    {
      title: "🐛 Bug Fixes",
      labels: ["bug", "Type:Bug"],
      bump: "patch",
    },
    {
      title: "📚 Documentation",
      labels: ["docs", "documentation", "Type:Documentation"],
      bump: "patch",
    },
    {
      title: "✅ Tests",
      labels: ["test", "tests", "Type:Test"],
      bump: "patch",
    },
    {
      title: "📦️ Dependencies",
      labels: ["dependency", "dependencies", "Type:Dependencies"],
      bump: "patch",
      weight: -1,
    },
  ],
  useVersionPrefixV: true,
  getLatestTag: ({ exec }) => {
    // the plugin uses the latest tag to determine the next version
    // and the changes that are included in the upcoming release.
    const branch = getBranch(exec);
    let tags = getTags(exec);

    if (branch.startsWith("stable-")) {
      const [_, majorAndMinor] = branch.split("-");
      // we only care about tags that are within the range of the current stable branch.
      // e.g. if the branch is stable-1.2, we only care about tags that are v1.2.x.
      const matchingTags = tags.filter((t) =>
        t.startsWith(`v${majorAndMinor}`)
      );

      if (matchingTags.length) {
        tags = matchingTags;
      }
    }

    return tags.pop() || "v0.0.0";
  },
  useLatestRelease: ({ exec, nextVersion }) => {
    // check if the release should be marked as latest release on GitHub.
    const tags = getTags(exec);
    const latestTag = tags.pop() || "v0.0.0";
    return compareVersions(latestTag, nextVersion) === -1;
  },
};

const parseVersion = (tag: string) => {
  const version = tag.startsWith("v") ? tag.slice(1) : tag;
  const [main, pre] = version.split("-");
  const [major, minor, patch] = main.split(".").map(Number);
  return { major, minor, patch, pre };
};

const getBranch = (exec: any): string => {
  return exec("git rev-parse --abbrev-ref HEAD", {
    silent: true,
  }).stdout.trim();
};

const getTags = (exec: any) => {
  exec("git fetch --tags", { silent: true });
  const tagsOutput = exec("git tag", { silent: true }).stdout.trim();
  const tags: string[] = tagsOutput ? tagsOutput.split("\n") : [];
  return tags.filter((tag) => tag.startsWith("v")).sort(compareVersions);
};

const compareVersions = (a: string, b: string) => {
  const va = parseVersion(a);
  const vb = parseVersion(b);

  if (va.major !== vb.major) {
    return va.major - vb.major;
  }
  if (va.minor !== vb.minor) {
    return va.minor - vb.minor;
  }
  if (va.patch !== vb.patch) {
    return va.patch - vb.patch;
  }

  if (va.pre && !vb.pre) {
    return -1;
  }
  if (!va.pre && vb.pre) {
    return 1;
  }

  if (va.pre && vb.pre) {
    return va.pre.localeCompare(vb.pre);
  }

  return 0;
};
