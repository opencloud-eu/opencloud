/*
Package kql provides the ability to work with kql queries.

Not every aspect of the spec is implemented yet.
The language support will grow over time if needed.

The following spec parts are supported and tested:
  - 2.1.2 AND Operator
  - 2.1.6 NOT Operator
  - 2.1.8 OR Operator
  - 2.1.12 Parentheses
  - 2.3.1 String Tokens
  - 2.3.5 Date Tokens
  - 3.1.11 Implicit Operator
  - 3.1.12 Parentheses
  - 3.1.2 AND Operator
  - 3.1.6 NOT Operator
  - 3.1.8 OR Operator
  - 3.2.3 Implicit Operator for Property Restriction
  - 3.3.1.1.1 Implicit AND Operator
  - 3.3.5 Date Tokens

A double quote inside a quoted value is written as two double quotes, `"say ""hi"""`
is the value `say "hi"`. Nothing else is special inside the quotes, a backslash
escapes nothing. An unquoted value can't contain a double quote.

References:
  - https://learn.microsoft.com/en-us/sharepoint/dev/general-development/keyword-query-language-kql-syntax-reference
  - https://learn.microsoft.com/en-us/openspecs/sharepoint_protocols/ms-kql/3bbf06cd-8fc1-4277-bd92-8661ccd3c9b0
  - https://msopenspecs.azureedge.net/files/MS-KQL/%5bMS-KQL%5d.pdf
*/
package kql
