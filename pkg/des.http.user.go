package pkg

import (
	// "encoding/json"
	"fmt"
	"net/url"
	// "reflect"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
// 	"github.com/go-playground/validator/v10" // go get github.com/go-playground/validator/v10
)

func InitializeDESUserRoutes(app, api *fiber.App) {
	api.Route("/user", func(router fiber.Router) {

		router.Post("/register", HandleRegisterUser)
		router.Post("/login", HandleLoginUser)
		router.Post("/refresh", DesAuth, HandleRefreshAccessToken)
		router.Post("/terminate", DesAuth, HandleTerminateUserSessions)
		router.Post("/logout", DesAuth, HandleLogoutUser)

		router.Get("/list", HandleGetUserList)

		app.Use("/ws", HandleWSUpgrade)
		router.Get("/ws", DesAuth, websocket.New(HandleUserSessionWS_Connect))
		// router.Get("/ws", DesAuth, HandleUserSessionWS_Request)
	})
}

/* AUTHENTICATE USER AND GET THEIR ROLE */
func DesAuth(c *fiber.Ctx) (err error) {

	authorization := c.Get("Authorization")
	// fmt.Printf("AUTHORIZATION: \n%s\n", authorization)
	// fmt.Printf("ACCESS_TOKEN: \n%s\n", c.Query("access_token"))

	tokenString := ""
	if strings.HasPrefix(authorization, "Bearer ") {
		tokenString = strings.TrimPrefix(authorization, "Bearer ")
	} else if c.Cookies("token") != "" {
		tokenString = c.Cookies("token")
	} else if c.Query("access_token") != "" {
		tokenString = c.Query("access_token")
	}
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).SendString("Authorization failed; please log in.")
	}

	claims, err := GetClaimsFromTokenString(tokenString)
	if err != nil {
		txt := fmt.Sprintf("Authorization failed; JWT claims error: %s", err.Error() )
		return c.Status(fiber.StatusUnauthorized).SendString(txt)
	}

	c.Locals("role", claims["rol"])
	c.Locals("sub", claims["sub"])

	return c.Next()
}
func ValidatePostRequestBody_UserResponse(c *fiber.Ctx, ur *UserResponse) (err error) {

	if err = ParseRequestBody(c, ur); err != nil { return }

	if !ValidateUUIDString(ur.ID.String()){ 
		err = fmt.Errorf("Invalid user ID: %s", ur.ID.String())
		return 
	}

	return
}
func ValidatePostRequestBody_UserSession(c *fiber.Ctx, us *UserSession) (err error) {

	if err = ParseRequestBody(c, us); err != nil { return }

	if !ValidateUUIDString(us.SID.String()){ 
		err = fmt.Errorf("Invalid session ID: %s", us.SID.String())
		return 
	}

	*us, err = UserSessionsMapRead(us.SID.String())

	return
}


func HandleWSUpgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

/* CREATE A NEW USER WITH DEFAULT ROLES */
func HandleRegisterUser(c *fiber.Ctx) (err error) {

	/* PARSE AND VALIDATE REQUEST DATA */
	runp := RegisterUserInput{}
	if err := c.BodyParser(&runp); err != nil {
		txt := fmt.Sprintf("Invalid request body: %s", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(txt)
	}

	if errors := ValidateStruct(runp); errors != nil {
		txt := fmt.Sprintf("Invalid request body: %v", errors)
		return c.Status(fiber.StatusBadRequest).SendString(txt)
	}

	if runp.Password != runp.PasswordConfirm {
		return c.Status(fiber.StatusBadRequest).SendString("Passwords do not match.")
	}

	/* CREATE A NEW USER WITH DEFAULT ROLES */
	user, err := RegisterUser(runp)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": user.FilterUserRecord()})
}

/* AUTHENTICATE USER INPUT AND RETURN JWTs */
func HandleLoginUser(c *fiber.Ctx) (err error) {

	/* PARSE AND VALIDATE REQUEST DATA */
	lunp := LoginUserInput{}
	if err := c.BodyParser(&lunp); err != nil {
		txt := fmt.Sprintf("Invalid request body: %s", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(txt)
	}

	if errors := ValidateStruct(lunp); errors != nil {
		txt := fmt.Sprintf("Invalid request body: %v", errors)
		return c.Status(fiber.StatusBadRequest).SendString(txt)
	}

	/* ATTEMPT LOGIN */
	us, err := LoginUser(lunp)
	if err != nil {
		txt := fmt.Sprintf("Login failed: %v", err)
		return c.Status(fiber.StatusBadGateway).SendString(txt)
	} // Json("HandleLoginUser( ) -> LoginUser( ) -> us: ", us)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"user_session": us})
}

/* VERIFY REFRESH TOKEN AND RETURN NEW ACCESS TOKEN */
func HandleRefreshAccessToken(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleRefreshAccessToken( )\n")

	/* PARSE AND VALIDATE REQUEST DATA */
	us := UserSession{}
	if err := c.BodyParser(&us); err != nil {
		txt := fmt.Sprintf("Invalid request body: %s", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(txt)
	} // Json("HandleRefreshAccessToken(): -> c.BodyParser(&us) -> user session", us)

	if err = us.RefreshAccessToken(); err != nil {
		txt := fmt.Sprintf("Authorization failed. %s", err.Error())
		return c.Status(fiber.StatusUnauthorized).SendString(txt)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"user_session": us})
} 
func HandleUserSessionWS_Connect(ws *websocket.Conn) {
	// fmt.Printf("\nHandleUserWSConnect( )\n")

	/* CHECK USER PERMISSION */
	if !UserRole_Viewer(ws.Locals("role")) {
		SendWSConnectionError(ws, "You must be a viewer to connect.")
		return
	}

	/* PARSE AND VALIDATE REQUEST DATA - SESSION ID */
	sid, err := url.QueryUnescape(ws.Query("sid"))
	if err != nil {
		SendWSConnectionError(ws, err.Error())
		return
	}

	if !ValidateUUIDString(sid) {
		SendWSConnectionError(ws, "Invalid user session ID.")
		return
	}

	/* GET UserSession FROM UserSessionsMap */
	us, err := UserSessionsMapRead(sid)
	if err != nil {
		SendWSConnectionError(ws, err.Error())
		return
	}

	/* CONNECT USER SESSION *** DO NOT RUN IN GO ROUTINE *** */
	us.UserSessionWS_Connect(ws)
}


func HandleLogoutUser(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleLogoutUser( )\n")

	/* PARSE AND VALIDATE REQUEST DATA */
	us := UserSession{}
	if err = ValidatePostRequestBody_UserSession(c, &us); err != nil { 
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}  // Json("HandleLogoutUser(): -> c.BodyParser(&us) -> user session", us)

	us.LogoutUser()

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "You have logged out."})
}

/* REVOKE ACCESS BASED ON A USER ID ( ALL SESSIONS ) */
func HandleTerminateUserSessions(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleTerminateUserSessions( )\n")

	/* CHECK USER PERMISSION */
	if !UserRole_Admin(c.Locals("role")) {
		txt := "You must be an administrator to revoke user access."
		return c.Status(fiber.StatusForbidden).SendString(txt)
	}

	ur := UserResponse{}
	/* PARSE AND VALIDATE REQUEST DATA */ 
	if err = ValidatePostRequestBody_UserResponse(c, &ur); err != nil { 
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	} // Json("HandleTerminateUserSessions( ) -> ValidatePostRequestData( ) -> ur", ur)


	txt := fmt.Sprintf("%d user sessions terminated.", TerminateUserSessions(ur))
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": txt})
}

/* RETURNS A LIST OF FILTERED USER RECORDS */
func HandleGetUserList(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleGetUserList( ):\n")

	userList, err := GetUserList()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"users": userList})
}
