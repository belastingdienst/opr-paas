---
title: Welcome!
summary: Introduction to contributing to the Paas operator.
authors:
  - devotional-phoenix-97
  - hikarukin
date: 2024-07-04
---

Contributing to Paas source code
================================

If you want to contribute to the source code of the Paas operator and innovate in
the database in Kubernetes space, this is the right place. Welcome!

We have a truly open source soul. That's why we welcome new contributors. Our
goal is to enable you to become the next committer of Paas, by having a good set
of docs that guide you through the development process. Having said this, we know
that everything can always be improved, so if you think our documentation is not
up to scratch, please let us know or provide a pull request based on your experience.

Development workflow
--------------------

The Paas operator project follows [trunk-based development](https://trunkbaseddevelopment.com),
with the `main` branch representing the trunk.

We adopt the ["Github Flow"](https://guides.github.com/introduction/flow/)
development workflow, with some customizations:

- the [Continuous Delivery](https://en.wikipedia.org/wiki/Continuous_delivery)
  branch is called `main` and is protected;
- Github is configured for linear development (no merge commits);
- development happens in separate branches created from the `main` branch and
  called "*dev/ISSUE_ID*";
- once completed, developers must submit a pull request;
- reviews by two different maintainers are required before a pull request can be
  merged;

We also adopted the [conventional commit](https://www.conventionalcommits.org/en/v1.0.0/)
format for commit messages.

The [roadmap](https://github.com/orgs/belastingdienst/projects/1) is defined as a
[Github Project](https://docs.github.com/en/issues/trying-out-the-new-projects-experience/about-projects).

Finally, we have an [operational Kanban board](https://github.com/orgs/belastingdienst/projects/2)
we use to organize the flow of items.

---

<!--
TODO:

- Add architecture diagrams in the "contribute" folder
- ...

-->

Testing the latest development snapshot
---------------------------------------

If you want to test or evaluate the latest development snapshot of Paas before
the next official patch release, you can simply run:

```sh
kubectl apply -f \
  https://raw.githubusercontent.com/belastingdienst/artifacts/main/manifests/Paas-operator-manifest.yaml
```

Your development environment for Paas
-------------------------------------

In order to write even the simplest patch for Paas you must have setup your
workstation to build and locally test the version of the operator you are developing.
All you have to do is follow the instructions you find in
["Setting up your development environment for Paas"](development-environment/index.md).

Submit a pull request
---------------------

> First and foremost: as a potential contributor, your changes and ideas are
> welcome at any hour of the day or night, weekdays, weekends, and holidays.
> Please do not ever hesitate to ask a question or send a PR.

**IMPORTANT:** before you submit a pull request, please read this document from
the Istio documentation which contains very good insights and best practices:
["Writing Good Pull Requests"](https://github.com/istio/istio/wiki/Writing-Good-Pull-Requests).

If you have written code for an improvement to Paas or a bug fix,
please follow this procedure to submit a pull request:

1. [Create a fork](development-environment/index.md#forking-the-repository) of Paas
2. Self-assign the ticket and begin working on it in the forked project. Move
   the ticket to `Analysis` or `In Development` phase of
   [Paas operator development](https://github.com/orgs/belastingdienst/projects/2)
3. [Run the e2e tests in the forked repository](e2e_testing_environment/README.md#running-e2e-tests-on-a-fork-of-the-repository)
4. Once development is finished, create a pull request from your forked project
   to the Paas project and move the ticket to the `Waiting for First Review`
   phase. Please make sure the pull request title and message follow
   [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)

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