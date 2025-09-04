How to Contribute
=================

Enmime highly encourages third-party patches. There is a great deal of MIME
encoded email out there, so it's likely you will encounter a scenario we
haven't.

### tl;dr:

- Please add a unit test for your fix or feature
- Ensure clean run of `make test lint`


## Getting Started

If you anticipate your issue requiring a large patch, please first submit a
GitHub issue describing the problem or feature. Attach an email that illustrates
the scenario you are trying to improve if possible. You are also encouraged to
outline the process you would like to use to resolve the issue. I will attempt
to provide validation and/or guidance on your suggested approach.


## Making Changes

Create a topic branch based on our `main` branch.

1. Make commits of logical units.
2. Add unit tests to exercise your changes.
3. **Scrub personally identifying information** from test case emails, and
   keep attachments short.
4. Ensure the code builds and tests with `make test`
5. Run the updated code through `make lint`


## Thanks

Thank you for contributing to enmime!
