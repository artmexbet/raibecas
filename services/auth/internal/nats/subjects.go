package nats

// NATS subjects for auth service

// Request/Reply subjects - auth service handles these
const (
	SubjectAuthRegister       = "auth.register"
	SubjectAuthLogin          = "auth.login"
	SubjectAuthRefresh        = "auth.refresh"
	SubjectAuthValidate       = "auth.validate"
	SubjectAuthLogout         = "auth.logout"
	SubjectAuthLogoutAll      = "auth.logout_all"
	SubjectAuthChangePassword = "auth.change_password"
)

// Event subjects - auth service publishes to these
const (
	SubjectAuthUserRegistered        = "auth.user.registered"
	SubjectAuthUserLogin             = "auth.user.login"
	SubjectAuthUserLogout            = "auth.user.logout"
	SubjectAuthPasswordReset         = "auth.password.reset"
	SubjectAuthRegistrationRequested = "auth.registration.requested"
)

// Event subjects - auth service subscribes to these (from admin service)
const (
	SubjectAdminRegistrationApproved = "admin.registration.approved"
	SubjectAdminRegistrationRejected = "admin.registration.rejected"
)
