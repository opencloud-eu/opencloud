---
title: 'End-to-End (E2E) Test Standards'
sidebar_position: 2
id: e2e-testing-standards
---



## Introduction

In OpenCloud, we use Playwright for webUI test automation. We benefit from lower barriers to entry, readability, and usability when test standards are consistent across repositories. For example:

- **Reusability:** Enhances reusability by making it easier to reuse functions, locators, shared steps, and other test code. Finding the necessary components and functionalities will be simple for someone working on a test resulting in reduced duplication and the requirement for rework after the initiation of a code review.

- **Facilitates reviews:** Code can be examined more quickly. Since you can quickly determine what the test code is doing, it eases the mental strain of code review.

- **Faster onboarding:** By following naming conventions, new contributors are onboarded more rapidly and feel comfortable enough to contribute to the codebase.

Here are the test standards and guidelines we adhere to when creating Playwright tests at OpenCloud.

## Folder Structure:

- `tests/:`

  - `e2e/`: Main folder containing all (end-to-end) E2E test-related files.

    - `cucumber/`: Main folder containing all Cucumber(BDD) test-related files.

      - `features/`: Contains Gherkin feature files.

        - `<test-suite-folder>/`: Collection house for "**related"** feature files.

          - `<aFeatureFile>.feature`: A feature file.

      - `steps/`: Holds the step definition files for mapping Gherkin steps to code.

        - `<stepDefinition>.ts`: Step definitions for each feature.

      - `hooks/`: Cucumber hooks for setting up and tearing down test environments.
        - `hooks.ts`: Contains `Before`, `After`, and other lifecycle hooks.

    - `support/`: Playwright (Test implementation)

      - `api/`: Contains API-related test files and configurations.

        - `<api-folder>/ `: Specific API tests for a particular service.

      - `objects/`: Contains the Page Object classes.

        - `<specific-page-object-folder>/`: Collection house for related page objects for each webpage or component.

          - `<individualPageObject>.ts`: Page Object for each webpage or component.

      - `utils/`: Utility functions and common helpers.

        - `helpers.ts`: Common utility functions (e.g., date formatting, data generation).

      - `test-data/`: Static test data files or folders for upload.

        - `filesForUpload/`: Static test data files for upload.

    - `config/`: Configuration files for Playwright and other tools.

      - `playwright.config.ts`: Playwright configuration.

    - `reports/`: Generated test reports (e.g., HTML, JSON).

      - `screenshots/`: Captured screenshots during test execution.

      - `videos/`: Recorded videos of test runs.

## Test Structure - Arrange, Act, Assert:

We can follow the AAA (Arrange, Act, Assert) pattern when structuring the tests. In most cases, the Arrange step can be included in a Before block(hook).
Consider including comments defining each section to ease readability.

```typescript
// arrange, set up the initial conditions for the test
// create a property
await createProperty()

// act, perform the action that you want to test
// raise a charge
// this could involve calling methods or functions defined in your page objects
await raiseCharge()

// assert, verify that the action had the expected outcome
// confirm charge has been raised
expect(charge).toBe('raised')
```

## Page Object Model (POM)

Every page should possess one POM file to enhance maintainability and scalability of our tests. It should include all the selectors and functions which are needed to interact with the UI page or component(s).

All interactions should be done using the page objects, no selectors in your tests.

All assertions should be in your test, no assertion in the POM

DO 👍

```typescript
// POM file './pageOobjects/foo/'
// add all locators and functions related to the page.
// allowing all tests to reuse

import { expect, Locator, Page } from '@playwright/test'

export class FooPage {
  readonly errorMessage: Locator

  constructor(page: Page) {
    this.page = page
    this.errorMessage = page.locator('.error-message')
  }
}

// test file './steps/foo.ts'
import { FooPage } from './pageObjects/foo'

let fooPage: FooPage

Then('error message should be visible', async function ({ page }) {
  const fooPage = new FooPage({ page })
  await expect(fooPage.errorMessage).toBeVisible()
})
```

DO NOT ⚔️

```typescript
// test file './steps/foo.ts'
// include locators directly in test
import { Locator, Page } from '@playwright/test'

Then('error message should be visible', async function ({ page }) {
  await expect(page.locator('.error-message')).toBeVisible()
})
```

## Waiting:

Playwright uses auto-waiting, so we avoid artificial waiting, the exception being if it is really necessary.

DO NOT ⚔️

```typescript
await page.waitForTimeout(5000)
```

This can cause flaky tests as we can rarely be certain the amount of wait time is enough. It can also unnecessarily increase the test run time. Instead, we can try:

DO 👍

```typescript
await page.goto(fooBarURL, {
  waitUntil: 'domcontentloaded'
})
```

DO 👍

```typescript
const element = page.locator('some-locator-path')
element.waitFor({ visible: true })
```

DO 👍

```typescript
await fooPage.buttonFoo.click()
await expect(fooPage.titlePage).toBeVisible()
```

## Selectors

Avoid selectors tied to implementation and page structure.

Instead, we can prioritize the below, based on [testing-library guiding principles](https://testing-library.com/docs/queries/about/#priority)

- `getByRole` (this aids accessibility, reflects how users and assistive technology perceive the page)

- `getByText`

- `getByTestId` (add them when needed)

DO NOT ⚔️

```javascript
page.locator('.opt-u > div > .summary > div:nth-child(4) > div')
```

DO 👍

```javascript
page.locator('#foo-button')
page.getByText('OK')
```

## Naming Conventions

### Files and folders

- **Files**: Declare in **_camelCase_**
- **Folders**: Declare in **_kebab-case_**

### Variables

Declare in **_camelCase_**.

### Booleans

Start with ‘is’, ‘has’, ‘are’, ‘have’. This helps spot that this is a boolean while skimming the code. Still declared in **_camelCase_**.

```typescript
let isTurnedOn = false
```

### Page Objects / Classes

Declare in **_PascalCase_**.

Use descriptive naming, which can help the reader quickly identify what page or page component this is covering. Use as much context as needed from your product to make the name meaningful.

DO NOT ⚔️

```typescript
export class newModal
```

DO 👍

```typescript
export class AddWorksOrderModal
```

### Locators

Use descriptive naming, which can help the reader quickly identify what element the locator is targeting.

As an example, you can use a naming structure that contains **_“action / name of element” + “type of element”_**.

**_Defining type of element_** - These are your basic HTML element types, they’ll be defined and named in the design system, or as a team you can align on a consistent naming of the elements. Example: checkbox, tickbox, button, tooltip

**_Defining action / name -_** Think about what action this element will perform when interacted with. Or any existing name/text of the element

DO 👍

```typescript
// This element is a submit button for the user registration form
const submitButton = await page.locator('<locator-path>')
```

DO 👍

```typescript
// This element is a button for uploading a profile picture
const uploadProfilePictureButton = await page.locator('<locator-path>')
```

### Function names

Always start function names with a **_“verb”_**, followed by the **_“component context”_** that the function is interacting with i.e. what entity it is having an effect on.

DO 👍

```typescript
getWorksOrder()
printTransactions()
deleteProperty()
```

## Gherkin Best Practices: Do's and Don'ts for Effective Features & Scenarios

### Writing Features

DO 👍

- **Be Descriptive:** Use clear and descriptive language that accurately reflects the functionality.
- **Keep It Concise:** Avoid overly long titles or descriptions. Aim for brevity while maintaining clarity.
- **Use Active Voice:** Write in an active voice to make it clear who is performing the action.
- **Contextual Information:** If applicable, provide context about the user role or the scenario to clarify who benefits from the feature.

DO NOT ⚔️

- **Avoid Vague Titles:** Titles like "Test Feature" or "Feature 1" do not provide meaningful information.
- **Neglect User Perspective:** Failing to mention the user role can make it unclear who the feature is intended for.

Example of a well-written feature:

```gherkin
Feature: Password Management for Registered Users
  As a registered user
  I want to set a new password
  So that I can secure my account
```

### Writing Scenarios

- **Use Clear and Descriptive Scenario Titles:** Ensure that each scenario title clearly conveys the action being tested and the expected outcome.

```gherkin
Scenario: User successfully registers with valid details
```

- **Use Clear Given/When/Then Steps:** Clearly define the context, action, and expected outcome(success or error messages if any).

```gherkin
Given the user is on the registration page
When the user enters valid details
Then the user should see a confirmation message "Registration successful! Welcome to our platform."
```

- **Use "tries to" for Negation:** This syntax is effective for scenarios where an action is expected to fail.

```gherkin
Scenario: User tries to log in with incorrect credentials
Given the user is on the registration page
When the user tries to log in with username "Alice" and password "wrongPassword"
Then the user should see an error message
  """
  Incorrect username or password.
  """
```

## Broad Gherkin Guidelines

This [OpenCloud developer manual](https://docs.opencloud.eu/server/next/developer_manual/testing/acceptance-tests.html#how-to-write-acceptance-tests) provides comprehensive guidelines and best practices for writing acceptance tests using the Gherkin syntax, a widely adopted language for defining test scenarios in a human-readable format. The manual outlines the specific syntax and structure required when crafting feature files and scenarios to ensure consistency and maintainability within the OpenCloud testing framework.

In OpenCloud, we use Playwright for webUI test automation. We benefit from lower barriers to entry, readability, and usability when test standards are consistent across repositories. For example:

- **Reusability:** Enhances reusability by making it easier to reuse functions, locators, shared steps, and other test code. Finding the necessary components and functionalities will be simple for someone working on a test resulting in reduced duplication and the requirement for rework after the initiation of a code review.

- **Facilitates reviews:** Code can be examined more quickly. Since you can quickly determine what the test code is doing, it eases the mental strain of code review.

- **Faster onboarding:** By following naming conventions, new contributors are onboarded more rapidly and feel comfortable enough to contribute to the codebase.

Here are the test standards and guidelines we adhere to when creating Playwright tests at OpenCloud.

## Folder Structure:

- `tests/:`

  - `e2e/`: Main folder containing all (end-to-end) E2E test-related files.

    - `cucumber/`: Main folder containing all Cucumber(BDD) test-related files.

      - `features/`: Contains Gherkin feature files.

        - `<test-suite-folder>/`: Collection house for "**related"** feature files.

          - `<aFeatureFile>.feature`: A feature file.

      - `steps/`: Holds the step definition files for mapping Gherkin steps to code.

        - `<stepDefinition>.ts`: Step definitions for each feature.

      - `hooks/`: Cucumber hooks for setting up and tearing down test environments.
        - `hooks.ts`: Contains `Before`, `After`, and other lifecycle hooks.

    - `support/`: Playwright (Test implementation)

      - `api/`: Contains API-related test files and configurations.

        - `<api-folder>/ `: Specific API tests for a particular service.

      - `objects/`: Contains the Page Object classes.

        - `<specific-page-object-folder>/`: Collection house for related page objects for each webpage or component.

          - `<individualPageObject>.ts`: Page Object for each webpage or component.

      - `utils/`: Utility functions and common helpers.

        - `helpers.ts`: Common utility functions (e.g., date formatting, data generation).

      - `test-data/`: Static test data files or folders for upload.

        - `filesForUpload/`: Static test data files for upload.

    - `config/`: Configuration files for Playwright and other tools.

      - `playwright.config.ts`: Playwright configuration.

    - `reports/`: Generated test reports (e.g., HTML, JSON).

      - `screenshots/`: Captured screenshots during test execution.

      - `videos/`: Recorded videos of test runs.

## Test Structure - Arrange, Act, Assert:

We can follow the AAA (Arrange, Act, Assert) pattern when structuring the tests. In most cases, the Arrange step can be included in a Before block(hook).
Consider including comments defining each section to ease readability.

```typescript
// arrange, set up the initial conditions for the test
// create a property
await createProperty()

// act, perform the action that you want to test
// raise a charge
// this could involve calling methods or functions defined in your page objects
await raiseCharge()

// assert, verify that the action had the expected outcome
// confirm charge has been raised
expect(charge).toBe('raised')
```

## Page Object Model (POM)

Every page should possess one POM file to enhance maintainability and scalability of our tests. It should include all the selectors and functions which are needed to interact with the UI page or component(s).

All interactions should be done using the page objects, no selectors in your tests.

All assertions should be in your test, no assertion in the POM

DO 👍

```typescript
// POM file './pageOobjects/foo/'
// add all locators and functions related to the page.
// allowing all tests to reuse

import { expect, Locator, Page } from '@playwright/test'

export class FooPage {
  readonly errorMessage: Locator

  constructor(page: Page) {
    this.page = page
    this.errorMessage = page.locator('.error-message')
  }
}

// test file './steps/foo.ts'
import { FooPage } from './pageObjects/foo'

let fooPage: FooPage

Then('error message should be visible', async function ({ page }) {
  const fooPage = new FooPage({ page })
  await expect(fooPage.errorMessage).toBeVisible()
})
```

DO NOT ⚔️

```typescript
// test file './steps/foo.ts'
// include locators directly in test
import { Locator, Page } from '@playwright/test'

Then('error message should be visible', async function ({ page }) {
  await expect(page.locator('.error-message')).toBeVisible()
})
```

## Waiting:

Playwright uses auto-waiting, so we avoid artificial waiting, the exception being if it is really necessary.

DO NOT ⚔️

```typescript
await page.waitForTimeout(5000)
```

This can cause flaky tests as we can rarely be certain the amount of wait time is enough. It can also unnecessarily increase the test run time. Instead, we can try:

DO 👍

```typescript
await page.goto(fooBarURL, {
  waitUntil: 'domcontentloaded'
})
```

DO 👍

```typescript
const element = page.locator('some-locator-path')
element.waitFor({ visible: true })
```

DO 👍

```typescript
await fooPage.buttonFoo.click()
await expect(fooPage.titlePage).toBeVisible()
```

## Selectors

Avoid selectors tied to implementation and page structure.

Instead, we can prioritize the below, based on [testing-library guiding principles](https://testing-library.com/docs/queries/about/#priority)

- `getByRole` (this aids accessibility, reflects how users and assistive technology perceive the page)

- `getByText`

- `getByTestId` (add them when needed)

DO NOT ⚔️

```javascript
page.locator('.opt-u > div > .summary > div:nth-child(4) > div')
```

DO 👍

```javascript
page.locator('#foo-button')
page.getByText('OK')
```

## Naming Conventions

### Files and folders

- **Files**: Declare in **_camelCase_**
- **Folders**: Declare in **_kebab-case_**

### Variables

Declare in **_camelCase_**.

### Booleans

Start with ‘is’, ‘has’, ‘are’, ‘have’. This helps spot that this is a boolean while skimming the code. Still declared in **_camelCase_**.

```typescript
let isTurnedOn = false
```

### Page Objects / Classes

Declare in **_PascalCase_**.

Use descriptive naming, which can help the reader quickly identify what page or page component this is covering. Use as much context as needed from your product to make the name meaningful.

DO NOT ⚔️

```typescript
export class newModal
```

DO 👍

```typescript
export class AddWorksOrderModal
```

### Locators

Use descriptive naming, which can help the reader quickly identify what element the locator is targeting.

As an example, you can use a naming structure that contains **_“action / name of element” + “type of element”_**.

**_Defining type of element_** - These are your basic HTML element types, they’ll be defined and named in the design system, or as a team you can align on a consistent naming of the elements. Example: checkbox, tickbox, button, tooltip

**_Defining action / name -_** Think about what action this element will perform when interacted with. Or any existing name/text of the element

DO 👍

```typescript
// This element is a submit button for the user registration form
const submitButton = await page.locator('<locator-path>')
```

DO 👍

```typescript
// This element is a button for uploading a profile picture
const uploadProfilePictureButton = await page.locator('<locator-path>')
```

### Function names

Always start function names with a **_“verb”_**, followed by the **_“component context”_** that the function is interacting with i.e. what entity it is having an effect on.

DO 👍

```typescript
getWorksOrder()
printTransactions()
deleteProperty()
```

## Gherkin Best Practices: Do's and Don'ts for Effective Features & Scenarios

### Writing Features

DO 👍

- **Be Descriptive:** Use clear and descriptive language that accurately reflects the functionality.
- **Keep It Concise:** Avoid overly long titles or descriptions. Aim for brevity while maintaining clarity.
- **Use Active Voice:** Write in an active voice to make it clear who is performing the action.
- **Contextual Information:** If applicable, provide context about the user role or the scenario to clarify who benefits from the feature.

DO NOT ⚔️

- **Avoid Vague Titles:** Titles like "Test Feature" or "Feature 1" do not provide meaningful information.
- **Neglect User Perspective:** Failing to mention the user role can make it unclear who the feature is intended for.

Example of a well-written feature:

```gherkin
Feature: Password Management for Registered Users
  As a registered user
  I want to set a new password
  So that I can secure my account
```

### Writing Scenarios

- **Use Clear and Descriptive Scenario Titles:** Ensure that each scenario title clearly conveys the action being tested and the expected outcome.

```gherkin
Scenario: User successfully registers with valid details
```

- **Use Clear Given/When/Then Steps:** Clearly define the context, action, and expected outcome(success or error messages if any).

```gherkin
Given the user is on the registration page
When the user enters valid details
Then the user should see a confirmation message "Registration successful! Welcome to our platform."
```

- **Use "tries to" for Negation:** This syntax is effective for scenarios where an action is expected to fail.

```gherkin
Scenario: User tries to log in with incorrect credentials
Given the user is on the registration page
When the user tries to log in with username "Alice" and password "wrongPassword"
Then the user should see an error message
  """
  Incorrect username or password.
  """
```

## Broad Gherkin Guidelines

This [OpenCloud developer manual](https://docs.opencloud.eu/server/next/developer_manual/testing/acceptance-tests.html#how-to-write-acceptance-tests) provides comprehensive guidelines and best practices for writing acceptance tests using the Gherkin syntax, a widely adopted language for defining test scenarios in a human-readable format. The manual outlines the specific syntax and structure required when crafting feature files and scenarios to ensure consistency and maintainability within the OpenCloud testing framework.
