package email

import "fmt"

// VerificationEmailBody returns the plain-text body for an email verification message.
func VerificationEmailBody(fullName, verifyURL string) string {
	return fmt.Sprintf(`Hi %s,

Welcome to Rekall! Please verify your email address by clicking the link below:

%s

This link expires in 24 hours. If you did not create an account, you can safely ignore this email.

— The Rekall Team
`, fullName, verifyURL)
}

// VerificationEmailSubject returns the subject line for a verification email.
func VerificationEmailSubject() string {
	return "Verify your Rekall email address"
}

// PasswordResetEmailBody returns the plain-text body for a password reset message.
func PasswordResetEmailBody(fullName, resetURL string) string {
	return fmt.Sprintf(`Hi %s,

We received a request to reset your Rekall password. Click the link below to set a new password:

%s

This link expires in 1 hour. If you did not request a password reset, you can safely ignore this email.

— The Rekall Team
`, fullName, resetURL)
}

// PasswordResetEmailSubject returns the subject line for a password reset email.
func PasswordResetEmailSubject() string {
	return "Reset your Rekall password"
}

// InvitationEmailBody returns the plain-text body for an organization invitation.
func InvitationEmailBody(orgName, inviterName, role, acceptURL string) string {
	return fmt.Sprintf(`Hi,

%s has invited you to join the "%s" organization on Rekall as a %s.

Accept the invitation by clicking the link below:

%s

This invitation expires in 7 days.

— The Rekall Team
`, inviterName, orgName, role, acceptURL)
}

// InvitationEmailSubject returns the subject line for an invitation email.
func InvitationEmailSubject(orgName string) string {
	return fmt.Sprintf("You've been invited to join %s on Rekall", orgName)
}
