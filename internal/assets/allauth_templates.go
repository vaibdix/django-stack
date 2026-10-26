package assets

import "strings"

type authPage struct {
	Name     string
	Title    string
	Heading  string
	Subtitle string
	Action   string
	Submit   string
	Footer   string

	Unless *authCondition
}

type authCondition struct {
	Variable string
	Heading  string
	Subtitle string
	Route    string
	Link     string
}

var AuthPages = []authPage{
	{
		Name:     "login",
		Title:    "Sign In",
		Heading:  "Sign in",
		Subtitle: "Welcome back. Enter your credentials to continue.",
		Action:   "account_login",
		Submit:   "Sign In",
		Footer:   `<a href="{% url 'account_reset_password' %}">Forgot password?</a> · Don't have an account? <a href="{% url 'account_signup' %}">Sign up</a>`,
	},
	{
		Name:     "signup",
		Title:    "Sign Up",
		Heading:  "Create your account",
		Subtitle: "It takes less than a minute.",
		Action:   "account_signup",
		Submit:   "Create Account",
		Footer:   `Already have an account? <a href="{% url 'account_login' %}">Sign in</a>`,
	},
	{
		Name:     "password_reset",
		Title:    "Reset Password",
		Heading:  "Reset your password",
		Subtitle: "We will email you a link to choose a new password.",
		Action:   "account_reset_password",
		Submit:   "Send reset link",
		Footer:   `<a href="{% url 'account_login' %}">Back to sign in</a>`,
	},
	{
		Name:     "password_reset_from_key",
		Title:    "Set a New Password",
		Heading:  "Choose a new password",
		Subtitle: "Pick something you have not used before.",
		Action:   "",
		Submit:   "Save password",
		Footer:   "",
		Unless: &authCondition{
			Variable: "token_fail",
			Heading:  "This link no longer works",
			Subtitle: "The password reset link is invalid, possibly because it has already been used.",
			Route:    "account_reset_password",
			Link:     "Request a new link",
		},
	},
	{
		Name:     "password_change",
		Title:    "Change Password",
		Heading:  "Change your password",
		Subtitle: "Enter your current password, then the new one.",
		Action:   "account_change_password",
		Submit:   "Save password",
		Footer:   "",
	},
}

var authNotices = []authPage{
	{
		Name:     "verification_sent",
		Title:    "Verify Your Email",
		Heading:  "Check your inbox",
		Subtitle: "We sent a verification link to the address you signed up with. Follow it to finish creating your account.",
	},
	{
		Name:     "password_reset_done",
		Title:    "Reset Link Sent",
		Heading:  "Check your inbox",
		Subtitle: "If an account exists for that address, a password reset link is on its way.",
	},
	{
		Name:     "password_reset_from_key_done",
		Title:    "Password Changed",
		Heading:  "Your password is set",
		Subtitle: "You can now sign in with your new password.",
	},
}

var djangoLoginPage = authPage{
	Name:     "login",
	Title:    "Sign In",
	Heading:  "Sign in",
	Subtitle: "Welcome back. Enter your credentials to continue.",
	Action:   "login",
	Submit:   "Sign In",
}

func DjangoLoginTemplate(tailwind bool) string {
	return authFormPageWith(djangoLoginPage, tailwind, true)
}

type authData struct {
	Page           authPage
	Style          authStyle
	DjangoRedirect bool
	Body           string
}

func authFormPageWith(page authPage, tailwind, djangoRedirect bool) string {
	data := authData{Page: page, Style: authStyleFor(tailwind), DjangoRedirect: djangoRedirect}
	body := strings.TrimSuffix(Render("auth_form_body.html.tmpl", data), "\n")
	if page.Unless != nil {
		body = "    " + strings.ReplaceAll(body, "\n", "\n    ")
	}
	data.Body = body
	return Render("auth_form.html.tmpl", data)
}

func authNoticeTemplate(page authPage, tailwind bool) string {
	return Render("auth_notice.html.tmpl", authData{Page: page, Style: authStyleFor(tailwind)})
}

func allAuthEmailConfirm(tailwind bool) string {
	return Render("auth_email_confirm.html.tmpl", authData{Style: authStyleFor(tailwind)})
}

func AllAuthTemplates(tailwind bool) map[string]string {
	pages := map[string]string{
		"logout.html":        AllAuthLogout(tailwind),
		"email_confirm.html": allAuthEmailConfirm(tailwind),
	}
	for _, page := range AuthPages {
		pages[page.Name+".html"] = AuthFormTemplate(page, tailwind)
	}
	for _, page := range authNotices {
		pages[page.Name+".html"] = authNoticeTemplate(page, tailwind)
	}
	return pages
}

func AuthFormTemplate(page authPage, tailwind bool) string {
	return authFormPageWith(page, tailwind, false)
}

func AllAuthLogout(tailwind bool) string {
	if tailwind {
		return Render("auth_logout_tailwind.html.tmpl", nil)
	}
	return Render("auth_logout_plain.html.tmpl", nil)
}

type authStyle struct {
	Open, Close                                              string
	Subtitle, Form, Field, Error, FieldError, Submit, Footer string
}

var tailwindAuthStyle = authStyle{
	Open: `    <section class="flex min-h-[50vh] items-center justify-center px-5 py-8">
        <div class="panel w-full max-w-[420px] p-6">`,
	Close: `        </div>
    </section>`,
	Subtitle:   "mb-6 text-body",
	Form:       "space-y-4",
	Field:      "space-y-1",
	Error:      "rounded-[6px] border border-danger px-3 py-2 text-danger",
	FieldError: "mt-1 text-[14px] text-danger",
	Submit:     "button button-primary w-full",
	Footer:     "mt-6 text-center text-body",
}

var plainAuthStyle = authStyle{
	Open: `    <div class="auth-container">
        <div class="auth-card">`,
	Close: `        </div>
    </div>`,
	Subtitle:   "auth-subtitle",
	Form:       "auth-form",
	Field:      "auth-field",
	Error:      "auth-error",
	FieldError: "auth-field-error",
	Submit:     "button button-primary auth-submit",
	Footer:     "auth-footer",
}

func authStyleFor(tailwind bool) authStyle {
	if tailwind {
		return tailwindAuthStyle
	}
	return plainAuthStyle
}
