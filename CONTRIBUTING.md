# OpenCloud Contribution Guidelines

First, thank you for taking the time to read this and your interest in contributing to OpenCloud!

The following is a set of guidelines suitable to most of the projects hosted in the [OpenCloud Organization](https://github.com/opencloud-eu).
These are mostly guidelines, not rules.

Use your best judgment and feel free to propose changes to this document in a [pull request](https://github.com/opencloud-eu/opencloud/pulls).

For simplicity reasons, this document mostly refers to the [opencloud repository](https://www.github.com/opencloud-eu/opencloud),
but it should be easily transferable to other (sub)projects.

#### Table Of Contents

[I don't want to read this whole thing, I just have a question](#i-dont-want-to-read-this-whole-thing-i-just-have-a-question)

[What to know before getting started](#what-to-know-before-getting-started)
*   [OpenCloud is hosted on GitHub](#opencloud-is-hosted-on-github)
*   [OpenCloud Company, Engineering Partners and Community](#opencloud-company-engineering-partners-and-community)
*   [Licensing and CLA](#licensing-and-cla)

[How to Contribute](#how-to-contribute)
*   [Help spreading the word](#help-spreading-the-word)
*   [Reporting Bugs](#reporting-bugs)
*   [Suggesting Enhancements](#suggesting-enhancements)
*   [Your First Code Contribution](#your-first-code-contribution)
*   [Pull Requests](#pull-requests)
*   [Documentation Contributions](#documentation-contributions)
*   [Internationalization](#internationalization)

[Styleguide](#styleguide)
*   [Commit Messages](#commit-messages)
*   [Branch Naming](#branch-naming)
*   [Golang Styleguide](#golang-styleguide)

  ## I don't want to read this whole thing I just have a question

> **Note:** Please don't file an issue to ask a question. You'll get faster results by using the resources below.

For general questions, please refer to [OpenCloud's FAQs](https://docs.opencloud.eu/docs/admin/resources/faq/) or check the [project page](https://github.com/opencloud-eu) for communication channels.

## What to know before getting started

### OpenCloud is hosted on GitHub

To effectively contribute to OpenCloud, you need a GitHub account. You can get that for free at [GitHub](https://github.com/join). You can find howtos on the internet, for example, [here](https://www.wikihow.com/Create-an-Account-on-GitHub).

For other ways of contributing, for example, with translations, other systems require you to have an account as well, for example [Transifex](https://www.transifex.com).

The OpenCloud project follows the strict GitHub workflow of development as briefly [described here](https://guides.github.com/introduction/flow/).

### OpenCloud Company, Engineering Partners and Community

OpenCloud is largely created by developers employed by the [OpenCloud company](https://opencloud.eu) (Germany) and by engineering partners who work full-time on related code, e.g. [REVA](https://github.com/opencloud-eu/reva/).

Because of that, development can move fast for people who can't spend a comparable amount of time contributing. That shouldn't scare anyone away: we're honored by every contribution, big or small, and do our best to listen, review and consider all changes that make sense for the project.

### Licensing and CLA

There is *no CLA* required for any of the public code of OpenCloud. The licenses are defined in every sub project.

## How to Contribute

There are many ways to contribute to open source projects, and all are equally valuable and appreciated.

### Help spreading the word

This way to contribute to the project cannot be overestimated:
People who talk about their experience with OpenCloud and help others with that are the key to the success of the project.

There are too many ways of doing that to line them up here, but examples are answering questions in any social media or in the [OpenCloud Matrix channel](https://matrix.to/#/#opencloud:matrix.org), writing blog posts etc. pp.

There is no formal guideline to this, just do it :-)

### Reporting Bugs

This section guides you through submitting a bug report for OpenCloud. Following these guidelines helps maintainers and the community understand your report :pencil:, reproduce the behavior :computer:, and find related reports :mag_right:.

Before creating a bug report:

*   **Make sure you're on a recent version.** Use the latest release or current master to reproduce the problem — that helps attract developer attention.
*   **Determine which [repository](https://github.com/opencloud-eu) the problem belongs to.**
*   **Search [existing issues](https://github.com/search?q=org%3Aopencloud-eu+type%3Aissue+&type=issues)** first. If an open issue matches and you have new information, comment on it instead of opening a new one — use reactions instead of "+1" comments.

> **Note:** If you find a **Closed** issue that seems like the same problem, open a new issue and link to the original.

Bugs are tracked as [GitHub issues](https://guides.github.com/features/issues/). Create one on the right repository and fill in [the template](https://github.com/opencloud-eu/opencloud/issues/new?Type%3ABug&template=bug_report.md), covering:

*   **A clear, descriptive title.**
*   **Exact reproduction steps**, from a user perspective (e.g. "I want to share pictures with Grandma") — say which client/method you used, not just what you did.
*   **Specific examples**: links, files, or copy/pasteable snippets in [Markdown code blocks](https://help.github.com/articles/markdown-basics/#multiple-lines).
*   **Observed vs. expected behavior.**
*   **Screenshots or GIFs** demonstrating the problem (e.g. [LICEcap](https://www.cockos.com/licecap/) on macOS/Windows, [silentcast](https://github.com/colinkeenan/silentcast) or [byzanz](https://github.com/GNOME/byzanz) on Linux).
*   **Browser dev tools output**, if it's a web issue.
*   **Context**: did it start recently (e.g. after an update)? Can you reproduce it in an older version? Is it reliably reproducible, and under what conditions?
*   **Your configuration/environment**, as requested by the template.

### Suggesting Enhancements

This section guides you through submitting an enhancement suggestion, from new features to minor improvements.

Before suggesting one:

*   **Check whether an existing extension or component already provides it.**
*   **[Search existing suggestions](https://github.com/search?q=+is%3Aissue+user%3Aopencloud)** — if found, comment or react instead of opening a new issue.

Enhancement suggestions are tracked as [GitHub issues](https://guides.github.com/features/issues/). Create one in the right repository and fill in [the template](https://github.com/opencloud-eu/opencloud/issues/new?template=feature_request.md), covering:

*   **A clear, descriptive title.**
*   **A step-by-step description** of the suggested enhancement.
*   **Specific examples**, as [Markdown code blocks](https://help.github.com/articles/markdown-basics/#multiple-lines) where relevant.
*   **Why it would be useful** to most OpenCloud users.
*   **Other projects/products** where this already exists.

Always try to explain the problem that you want to see solved, and why it hurts today. That opens the space for a creative solution finding.
Consider if your proposal affects many users and if it is worth to maintain it for the time being. Programming something is cheap, maintaining it is expensive.

### Size of the Contribution

Before submitting a PR to any repository of OpenCloud, it is important to think about its size and fit to the project. That is a big factor how maintainers look at it, especially in times of AI assisted development.

#### Bug Fixes
Bug fixes in form of "one-liners", typo fixes, fixes to translations and such are always very appreciated. Never think "that is too trivial to submit", it is not.

#### Refactorings
Bigger changes that fix misbehaviour, refactor parts of the codebase are also appreciated, but be careful. Have tests and keep changes small. Possibly ask before submitting what the maintainers think of your idea.

#### Feature Additions
Yes, we love it, but we also apply the rules here, as we take the responsibility to maintain code that we include into the project. Try to structure the changes you plan into multiple steps to make it easier to understand and review. Try to consider alternatives and document your decisions (ADR).

Most important: make sure to talk before you invest work and tokens. There is a chance that we will not take your contribution even if it is formally great because it does not fit the roadmap.

#### "Scratch your itch"
"Scratching your own itch" and pushing the results upstream is in general a great motivation to contribute to open source projects. OpenCloud supports that idea.
However, we have to keep the main direction of the project in mind, so we can not accept every "special purpose" feature.

Make sure to find a balance between functions that are good for everybody and your own needs. You will have to keep private patches for some of your additions. OpenCloud comes with a [web extension system](https://docs.opencloud.eu/de/docs/dev/web/extension-system/) to make independently maintained extensions easy.

### Your First Code Contribution

Unsure where to begin contributing to OpenCloud? You can start by looking through these `Needs-help` issues:

*   The [Good first issue](https://github.com/opencloud-eu/opencloud/labels/Type%3Agood-first-issue) label marks good items to start with.
*   The [Feature Request](https://github.com/opencloud-eu/opencloud/issues?q=state%3Aopen%20label%3AType%3AFeature-Request) label lists features the community would like to see implemented.

It is fine to pick one of the lists following personal preference.
While not perfect, the number of comments is a reasonable proxy for the impact a given change will have.

If you want to work on that is not yet covered by an issue, make sure to create one before you start to spend your time on implementing it. That will give you an indication if your contribution will be accepted.

Think about creating an extension for OpenCloud rather than trying to push it to the core. That makes it easier to distribute maintenance over more shoulders.

To find out how to set up OpenCloud for local development, please refer to the [Developer Documentation](https://docs.opencloud.eu/docs/dev/web/getting-started) for the web side, and the general server [README](https://github.com/opencloud-eu/opencloud/blob/main/README.md) for backend setup. Both contain information that will come in handy when starting to work on the project.

### Pull Requests

All contributions to OpenClouds projects use pull requests following the [GitHub PR workflow](https://guides.github.com/introduction/flow/).

Please follow these steps to have your contribution considered by the maintainers:

*   Follow all instructions in [the template](https://github.com/opencloud-eu/opencloud/blob/main/.github/pull_request_template.md)
*   Follow the [styleguide](#styleguide) where applicable
*   After you submit your pull request, verify that all [status checks](https://help.github.com/articles/about-status-checks/) are passing <details><summary>What if the status checks are failing?</summary>If a status check is failing, and you believe that the failure is unrelated to your change, please leave a comment on the pull request explaining why you believe the failure is unrelated. A maintainer will re-run the status check for you. If we conclude that the failure was a false positive, then we will open an issue to track that problem with our status check suite.</details>

While the prerequisites above must be satisfied prior to having your pull request reviewed, the reviewer(s) may ask you to complete additional design work, tests, or other changes before your pull request can be ultimately accepted.

Please be patient. Silence does not mean denail, but sitting in a queue. While waiting, consider to review other PRs to speed up the process.

### Documentation Contributions

OpenCloud is very proud of the documentation it has, which is the work of a great team of people. Of course, also the documentation is open to contributions.

You find more guidance in the [Documentation Repo](https://github.com/opencloud-eu/docs) on how to get started.

### Internationalization

Our projects are getting translated into many languages to allow people from all over the world to use OpenCloud in their native language.
For translations, OpenCloud uses [Transifex](https://www.transifex.com) as a community-based collaboration platform for internationalization.

For contributions please refer to the [Transifex Resources](https://www.transifex.com/resources/) to learn how to improve OpenClouds translations there.

## Styleguide

To keep up with a consistent code and tooling landscape, some OpenCloud modules maintain styleguide for contributions.
It is mandatory to follow them in contributions.

### Commit Messages

*   Use the present tense ("Add feature" not "Added feature")
*   Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
*   Limit the first line to 72 characters or less
*   Reference issues and pull requests liberally after the first line
*   When only changing documentation, include `[docs-only]` in the commit title
*   Use conventional commit messages, see https://www.conventionalcommits.org/en/v1.0.0/

### Branch Naming

*   Use short, descriptive, hyphen-separated lowercase names (e.g. `add-new-feature`, not `add_new_feature` or `bugfix123`).
*   Avoid special characters or spaces; keep names concise, ideally under 30 characters.
*   Consider including the issue number for easy reference, e.g. `issue-45-fix-login-bug`.

### Golang Styleguide

Use the built-in golang code formatter before submitting the patch.
Also, consulting documentation like [Effective Go](https://golang.org/doc/effective_go) or [Practical Go](http://bit.ly/gcsg-2019) helps to improve the code quality.
