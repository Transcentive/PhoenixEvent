# LoginForm

The login panel: a Username field, a Password field with Show/Hide, a "Forgot password?" link and one primary button. Class: `pe-login` on the `form`, composed from TextField, PasswordField and Button.

You provide: the organization name shown under the title, the form action, and the reset link target.

- Exactly two fields: Username and Password. An email address is accepted in Username.
- The organization name under the title shows which organization the session will belong to. How a person picks the organization is an open decision in the specification, so this panel displays the name and leaves the picker out.
- A failed login shows one message above the fields that covers both fields together, so the response stays the same for every account.
