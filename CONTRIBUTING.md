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

[Additional Notes](#additional-notes)
*   [Issue and Pull Request Labels](#issue-and-pull-request-labels)

  ## I don't want to read this whole thing I just have a question

> **Note:** Please don't file an issue to ask a question. You'll get faster results by using the resources below.

For general questions, please refer to [OpenCloud's FAQs](https://opencloud.eu/faq/) or check the [project page](https://github.com/opencloud-eu) for communication channels.

## What to know before getting started

### OpenCloud is hosted on GitHub

To effectively contribute to OpenCloud, you need a GitHub account. You can get that for free at [GitHub](https://github.com/join). You can find howtos on the internet, for example, [here](https://www.wikihow.com/Create-an-Account-on-GitHub).

For other ways of contributing, for example, with translations, other systems require you to have an account as well, for example [Transifex](https://www.transifex.com).

The OpenCloud project follows the strict GitHub workflow of development as briefly [described here](https://guides.github.com/introduction/flow/).

### OpenCloud Company, Engineering Partners and Community

OpenCloud is largely created by developers who are employed by the [OpenCloud company](https://opencloud.eu), which is located in Germany.
It is providing support for OpenCloud for customers mainly in the EU. In addition, there are engineering partners who also work full-time on OpenCloud related code, for example, on the component [REVA](https://github.com/cs3org/reva/).

Because of that fact, the pace that the development is moving forward is sometimes high for people who are not willing and/or able to spend a comparable amount of time to contribute.
Even though this can be a challenge, it should not scare anybody away. Here is our clear commitment that we feel honored by everybody who is interested in our work and improves it, no matter how big the contribution might be.

We as the full-time devs from either organization are doing our best to listen, review and consider all changes that are brought forward following this guideline and make sense for the project.

### Licensing and CLA

 There is *no CLA* required for any of the public code of OpenCloud.

## How to Contribute

There are many ways to contribute to open source projects, and all are equally valuable and appreciated.

### Help spreading the word

This way to contribute to the project cannot be overestimated:
People who talk about their experience with OpenCloud and help others with that are the key to the success of the project.

There are too many ways of doing that to line them up here, but examples are answering questions in any social media or in the [OpenCloud Matrix channel](https://matrix.to/#/#opencloud:matrix.org), writing blog posts etc. pp.

There is no formal guideline to this, just do it :-)

### Reporting Bugs

This section guides you through submitting a bug report for OpenCloud. Following these guidelines help maintainers and the community understand your report :pencil:,
reproduce the behavior :computer: :computer:, and find related reports :mag_right:.

Before creating bug reports, please check [this list](#before-submitting-a-bug-report) as you might find out that you don't need to create one.
When you are creating a bug report, please [include as many details as possible](#how-to-submit-a-good-bug-report). Fill out [the required template](https://github.com/opencloud-eu/opencloud/issues/new?Type%3ABug&template=bug_report.md), the information it asks for helps to resolve issues faster.

> **Note:** If you find a **Closed** issue that seems like it is the same thing that you're experiencing, open a new issue and include a link to the original issue in the body of your new one. If you have permission to reopen the issue, feel free to do so.

#### Before Submitting A Bug Report

*   **Make sure you are running a recent version** Usually, developers' interest in old versions of software drops very fast once a new version has been released. So the general requirement is: Use the latest released version or even the current master to reproduce problems that you might encounter. That helps a lot to attract developers attention.
*   **Determine which [repository](https://github.com/opencloud-eu) the problem should be reported in**.
*   **Perform a [cursory search](https://github.com/search?q=org%3Aopencloud-eu+type%3Aissue+&type=issues)** with possibly a more granular filter on the repository to see if the problem has already been reported. If it has **and the issue is still open**, add a comment to the existing issue instead of opening a new one **if you have new information**. Please abstain from adding "+1" comments. Instead, use the GitHub reaction emojis to indicate that you are affected by the issue as well.

#### How to Submit A (Good) Bug Report

Bugs are tracked as [GitHub issues](https://guides.github.com/features/issues/). After you've determined [which repository](https://github.com/opencloud-eu) your bug is related to, create an issue on that repository and provide the following information by filling in [the template](https://github.com/opencloud-eu/opencloud/issues/new?Type%3ABug&template=bug_report.md).

Explain the problem and include additional details to help maintainers reproduce the problem:

*   **Use a clear and descriptive title** for the issue to identify the problem.
*   **Describe the exact steps which reproduce the problem** in as many details as possible. Start with describing, from a user perspective, what you tried to achieve, i.e. "I want to share some pictures with Grandma". When listing steps, **don't just say what you did, but explain how you did it**. For example, if you uploaded a file to OpenCloud, say which client you used, which way of uploading you chose, if the name was special somehow and how big it was.
*   **Provide specific examples to demonstrate the steps**. Include links to files or GitHub projects, or copy/pasteable snippets, which you use in those examples. If you're providing snippets in the issue, use [Markdown code blocks](https://help.github.com/articles/markdown-basics/#multiple-lines).
*   **Describe the behavior you observed after following the steps** and point out what exactly is the problem with that behavior.
*   **Explain which behavior you expected to see instead and why.**
*   **Include screenshots and animated GIFs** which show you following the described steps and clearly demonstrate the problem. You can use [this tool](https://www.cockos.com/licecap/) to record GIFs on macOS and Windows, and [this tool](https://github.com/colinkeenan/silentcast) or [this tool](https://github.com/GNOME/byzanz) on Linux.
*   **If you report a web browser related problem**, consider to using the browser's Web developer tools (such as the debugger, console or network monitor) to check what happened. Make sure to add screenshots of the utilities if you are short of time to interpret it.
*   **If the problem wasn't triggered by a specific action**, describe what you were doing before the problem happened and share more information using the guidelines below.

Provide more context by answering these questions:

*   **Did the problem start happening recently** (e.g. after updating to a new version) or was this always a problem?
*   If the problem started happening recently, **can you reproduce the problem in an older version?** What's the most recent version in which the problem doesn't happen? You can find more information about how to set up [test environments](https://docs.opencloud.eu/devel/testing) in the [developer documentation](https://docs.opencloud.eu/docs/dev/intro).
*   **Can you reliably reproduce the issue?** If not, provide details about how often the problem happens and under which conditions it normally happens.

Include details about your configuration and environment as asked for in the template.

### Suggesting Enhancements

This section guides you through submitting an enhancement suggestion for OpenCloud, including completely new features and minor improvements to existing functionality.
Following these guidelines help maintainers and the community understand your suggestion :pencil: and find related suggestions :mag_right:.

Before creating enhancement suggestions, please check [this list](#before-submitting-an-enhancement-suggestion) as you might find out that you don't need to create one.
When you are creating an enhancement suggestion, please [include as many details as possible](#how-to-submit-a-good-enhancement-suggestion).
Fill in [the template](https://github.com/opencloud-eu/opencloud/issues/new?template=feature_request.md), including the steps that you imagine you would take if the feature you're requesting existed.

#### Before Submitting An Enhancement Suggestion

*   **Check if there's already an extension or other component that provides that enhancement, even differently.**
*   **Perform a [cursory search](https://github.com/search?q=+is%3Aissue+user%3Aopencloud)** to see if the enhancement has already been suggested. If it has, add a comment to the existing issue instead of opening a new one. Feel free to use the GitHub emojis to indicate that you are in favor of an enhancement request.

#### How to Submit A (Good) Enhancement Suggestion

Enhancement suggestions are tracked as [GitHub issues](https://guides.github.com/features/issues/). After you've determined [which repository](https://github.com/opencloud-eu) your enhancement suggestion is related to, create an issue on that repository and provide the following information:

*   **Use a clear and descriptive title** for the issue to identify the suggestion.
*   **Provide a step-by-step description of the suggested enhancement** in as many details as possible.
*   **Provide specific examples to demonstrate the steps**. Include copy/pasteable snippets which you use in those examples, as [Markdown code blocks](https://help.github.com/articles/markdown-basics/#multiple-lines).
*   **Explain why this enhancement would be useful** to most OpenCloud users.
*   **List some other projects or products where this enhancement exists.**

### Your First Code Contribution

Unsure where to begin contributing to OpenCloud? You can start by looking through these `Needs-help` issues:

*   The [Good first issue](https://github.com/opencloud-eu/opencloud/labels/Topic%3Agood-first-issue) label marks good items to start with.
*   [Tests needed](https://github.com/opencloud-eu/opencloud/labels/Interaction%3ANeeds-tests) - issues which would benefit from a test.
*   [Help wanted issues](https://github.com/opencloud-eu/opencloud/labels/Interaction%3ANeeds-help) - issues which should be a bit more involved.

It is fine to pick one of the lists following personal preference.
While not perfect, the number of comments is a reasonable proxy for the impact a given change will have.

To find out how to set up OpenCloud for local development, please refer to the [Developer Documentation](https://docs.opencloud.eu/docs/dev/web/getting-started).
It contains a lot of information that will come in handy when starting to work on the project.

### Pull Requests

All contributions to OpenClouds projects use so-called pull requests following the [GitHub PR workflow](https://guides.github.com/introduction/flow/).

Please follow these steps to have your contribution considered by the maintainers:

*   Follow all instructions in [the template](https://github.com/opencloud-eu/opencloud/blob/main/.github/pull_request_template.md)
*   Follow the [styleguide](#styleguide) where applicable
*   After you submit your pull request, verify that all [status checks](https://help.github.com/articles/about-status-checks/) are passing <details><summary>What if the status checks are failing?</summary>If a status check is failing, and you believe that the failure is unrelated to your change, please leave a comment on the pull request explaining why you believe the failure is unrelated. A maintainer will re-run the status check for you. If we conclude that the failure was a false positive, then we will open an issue to track that problem with our status check suite.</details>

While the prerequisites above must be satisfied prior to having your pull request reviewed, the reviewer(s) may ask you to complete additional design work, tests, or other changes before your pull request can be ultimately accepted.

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

* Use short, descriptive names for your branches. For example, use `fix-login-bug` instead of `bugfix123`.
* Use hyphens to separate words in branch names. For example, use `add-new-feature` instead of `add_new_feature`.
* Avoid using special characters or spaces in branch names.
* Consider including the issue number in the branch name for easy reference. For example, use `issue-45-fix-login-bug` if the branch addresses issue #45.
* Keep branch names concise and to the point, ideally under 30 characters.
* Use lowercase letters to maintain consistency and avoid confusion.

### Golang Styleguide

Use the built-in golang code formatter before submitting the patch.
Also, consulting documentation like [Effective Go](https://golang.org/doc/effective_go) or [Practical Go](http://bit.ly/gcsg-2019) helps to improve the code quality.

## Additional Notes

### Issue and Pull Request Labels

This section lists the labels we use to help us track and manage issues and pull requests. Most labels are used across all OpenCloud repositories, but some are specific.

[GitHub search](https://help.github.com/articles/searching-issues/) makes it easy to use labels for finding groups of issues or pull requests you're interested in.
To help you find issues and pull requests, each label can be used in search links for finding open items with that label in the OpenCloud repositories.

The labels are loosely grouped by their purpose, but it's not required that every issue has a label from every group or that an issue can't have more than one label from the same group.

The list here contains all the more general categories of issues which are followed by a colon and a specific value.
For example, severity 1 looks like `Severity:sev1-critical`.

#### Platform

Describes the platform the issue is happening on, i.e. iOS or Windows.

#### Estimation

T-Shirt sizes for effort estimation to fix that bug or implement an enhancement. Ranges from XS to XXXL.

#### Priority

P1 to P4 (lowest) to indicate a priority. Mostly a tool for internal project management and support.

#### QA

Flags to indicate the internal QA status in terms of process and priority. Please leave alone unless you're QA ;-)

#### Severity

Severity for the product, mostly impacts on the user.

#### Type

The issue type helps to structure the issues in the agile categories (Epic, Story...) but also organizational ones.

#### Topic

A general category of the topic of a ticket.

#### Category

Categorizes the issue to also indicate the type of the issue.

#### Status

The status in the ticket life cycle. Keep an eye on that one, especially for the `Waiting-for-Feedback` tag which might indicate that the reporter is asked for feedback.

#### Interaction

Another label that indicates the type of the issue.

#### Browser

Important for browser-dependent web issues. It specifies the browser that shows the error.

#### Early-Adopter

Tags issues reported by one of the OpenCloud early adopters, i.e. customers and users who start using OpenCloud before its general availability.
