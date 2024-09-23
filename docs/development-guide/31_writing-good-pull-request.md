---
title: Writing Good Pull Requests
summary: How to write good Pull Requests
authors:
  - Istio Project
date: 2024-09-23
---

!!! Note
    The following text was shamelessly copied in full from the Istio project.
    The original lives at: [https://github.com/istio/istio/wiki/Writing-Good-Pull-Requests](https://github.com/istio/istio/wiki/Writing-Good-Pull-Requests)

    The reason for including the text here, is that we really liked the content
    and did not want to run the risk of having a broken link / losing the content
    in the future should the original ever be moved or deleted.

One of the biggest bottlenecks we have in Istio is PR reviews. By creating good PRs, you can help the reviewers go through your PR easily and get it checked-in quickly. This is a set of guidelines for creating good pull requests.

- [Communicate beforehand](#communicate-beforehand-ie-why-are-you-doing-this)
    - [Open a tracking issue](#open-a-tracking-issue)
    - [Use work-in-progress PRs for early feedback](#use-work-in-progress-prs-for-early-feedback)
- [Add a good explanation](#add-a-good-explanation-ie-what-exactly-are-you-doing)
- [Keep it short](#keep-it-short)
- [Organize into commits](#organize-into-commits)
- [Add tests!](#add-tests-ie-does-it-actually-work)
- [Tracking future work](#tracking-future-work)

## Communicate beforehand (i.e. why are you doing this?)

It's awful when a reviewer rejects your PR, or objects to your design. You now have to throw away your carefully crafted and authored PR and start over.

If you communicate your intent for the change to the reviewer beforehand and agree on the design, there is much less of a chance of outright rejection of PRs, or substantial change requests. It also gives an opportunity for the reviewer to think about the problem and think about how the system behavior would change, before they start reviewing the code.

### Open a tracking issue

Unless the PR is trivial, it is a good idea to open a bug to track the issue first. Especially if this is for fixing a bug. This allows capturing more detailed analysis (and repro steps if this is a bug) separately. It can also be used to track multiple PRs against the same problem.

### Use work-in-progress PRs for early feedback

A good way to communicate before investing too much time is to create a "Work-in-progress" PR and share it with your reviewers. The standard way of doing this is to add a "WIP:" prefix in your PR's title. This will let people looking at your PR know that it is not well baked yet. Our infrastructure also understands that the PR is not ready for merging yet, and will not allow accidental merging.

## Add a good explanation (i.e. what exactly are you doing?)

If you just write a cryptic title and nothing else, there is not much to go with for the reviewer. The reviewer will need to reconstruct what you're trying to accomplish from your code, which is not an easy task.

Writing a good, short, to the point explanation of what is going in your commits is extremely useful for the reviewer. If there are multiple things going on (i.e. a needed refactoring, a trivial bug fix you happened to catch along the way, an issue that you found for which you're adding a TODO for), add these as bullet-points in your PR description.

This not only helps the reviewer, but also people that are looking at the repo history, trying to figure out what has changed in a particular pull request.

**Do:**

```md
Subject: Fix bug(#449) that causes Foo component to crash when flag is not set.
Description:
+ This is caused by an off-by-one failure during iteration of nukes to launch.
+ Also fixed a race condition by adding a lock on the trigger mechanism that caused concurrent launches that caused a crash in the silo.
+ Adding a TODO for refactoring the code as well, as the cold war is over and we don't need this particular
defense mechanism anymore.
```

**Don't**

```md
Subject: Fix minor bug.
Description:
```

## Keep it short

The shorter the PR, the easier to review. Reviewing a PR requires the reviewer to understand how the system behavior is being changed. With bigger PRs, this becomes harder to understand, especially with the diff based nature of the review tools.

Keep your PRs as short as possible. A good rule of thumb is that if you PR ends up touching more than 500 lines, you should considering breaking it up into smaller PRs.

If there are refactorings that you've decided to do along the way, move them to a separate PR so that real changes aren't mingled with no-brainer refactoring changes.

## Organize into commits

If you must merge large changes, all in one go, then consider splitting your changes into multiple commits within the same pull request. This allows the reviewer to compartmentalize your changes and review them in isolation.

When making changes that are requested by your reviewers, add them as additional commits, instead of squashing it into the original. This allows the reviewer to quickly spot that his feedback is being incorporated.

## Add tests! (i.e. does it actually work?)

Whatever the issue you're fixing, add tests! Adding tests are the best way to convince the reviewer that what you're doing actually works. It also makes sure that the product will not regress and the issue will not occur again.

It is crucial to add the right type of tests. If the issue you are fixing is due to a particular library (e.g. cache library race condition causing cache poisoning), it is perfectly reasonable to write a small unit-test to avoid the regression.

However, if the issue is larger scoped (e.g. change in product behavior), then it is important to add the right integration or end-to-end test to verify the behavior.

## Tracking future work

Sometimes in the course of a PR review, a reviewer will point out more work that should be done as part of the change. When this happens, it is common for the author of the PR to say "Good idea, I'll do that in a follow-up PR". As part of this, it's generally desirable for the author to open a new issue (assuming there isn't one already) to track this extra work. The issue # should be included in the original PR so that the reviewer can rest assured that the work will not be forgotten.