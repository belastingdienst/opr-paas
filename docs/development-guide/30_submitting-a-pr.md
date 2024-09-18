---
title: Submitting a Pull Request
summary: How to submit a Pull Request
authors:
  - devotional-phoenix-97
  - hikarukin
date: 2024-07-04
---

> First and foremost: as a potential contributor, your changes and ideas are
> welcome at any hour of the day or night, weekdays, weekends, and holidays.
> Please do not ever hesitate to ask a question or send a PR.

!!! Tip
    Before you submit a pull request, please read this document from the Istio
    documentation which contains very good insights and best practices:
    ["Writing Good Pull Requests"](https://github.com/istio/istio/wiki/Writing-Good-Pull-Requests).

If you have written code for an improvement to Paas or a bug fix, please follow
this procedure to submit a pull request:

1. Create a fork of the Paas Operator project;
2. Add a comment to the related issue to let us know you're working on it;
3. Develop your feature or fix on your forked repository;
3. Run the e2e tests in your forked repository, see our [related e2e testing]](development-guide/e2e-tests/)
   documentation;
4. Once development is finished, create a pull request from your forked project
   to the Paas project.
   Please make sure the pull request title and message follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)

One of the maintainers will then proceed with the first review and approve the
CI workflow to run in the Paas project.  The second reviewer will run
end-to-end test against the changes in fork pull request. If testing passes,
the pull request will be labeled with `ok-to-merge` and will be ready for
merge.

Sign your work
--------------

We use the Developer Certificate of Origin (DCO) as an additional safeguard for
the Paas project. This is a well established and widely used mechanism to assure
contributors have confirmed their right to license their contribution under the
project's license.

Please read [https://developercertificate.org](https://developercertificate.org).

If you can certify it, then just add a line to every git commit message:

```
  Signed-off-by: Random J Developer <random@developer.example.org>
```

or use the command `git commit -s -m "commit message comes here"` to sign-off on your commits.

Use your real name (sorry, no pseudonyms or anonymous contributions).
If you set your `user.name` and `user.email` git configs, you can sign your
commit automatically with `git commit -s`.

You can also use git [aliases](https://git-scm.com/book/en/v2/Git-Basics-Git-Aliases)
like `git config --global alias.ci 'commit -s'`. Now you can commit with `git ci`
and the commit will be signed.