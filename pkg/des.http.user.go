package pkg

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func InitializeDESUserRoutes(app, api *fiber.App) {
	api.Route("/user", func(router fiber.Router) {

		router.Post("/register", HandleRegisterUser)
		router.Post("/login", HandleLoginUser)
		router.Post("/refresh", DesAuth, HandleRefreshAccessToken)
		router.Post("/terminate", DesAuth, HandleTerminateUserSessions)
		router.Post("/logout", DesAuth, HandleLogoutUser)

		router.Get("/list", GetUserList) /* TODO: AUTH */
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "fail",
			"message": "You are logged out.",
		})
	}

	claims, err := GetClaimsFromTokenString(tokenString)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}

	c.Locals("role", claims["rol"])
	c.Locals("sub", claims["sub"])

	return c.Next()
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
	// var payload *SignUpInput
	runp := RegisterUserInput{}
	if err := c.BodyParser(&runp); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	errors := ValidateStruct(runp)
	if errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "fail",
			"errors": errors,
		})
	}
	if runp.Password != runp.PasswordConfirm {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": "Passwords do not match",
		})
	}

	/* CREATE A NEW USER WITH DEFAULT ROLES */
	user, err := RegisterUser(runp)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   fiber.Map{"user": user.FilterUserRecord()},
	})
}

/* AUTHENTICATE USER INPUT AND RETURN JWTs */
func HandleLoginUser(c *fiber.Ctx) (err error) {

	/* PARSE AND VALIDATE REQUEST DATA */
	lunp := LoginUserInput{}
	if err := c.BodyParser(&lunp); err != nil {
		fmt.Println("SignInUser(c *fiber.Ctx) -> c.BodyParser")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("Malformed request body: %v", err.Error()),
		})
	}
	if errors := ValidateStruct(lunp); errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("Malformed request body: %v", errors),
		})
	}

	/* ATTEMPT LOGIN */
	us, err := LoginUser(lunp)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("Login failed: %v", err),
		})
	} // Json("HandleLoginUser( ) -> LoginUser( ) -> us: ", us)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":       "success",
		"user_session": us,
	})
}

/* VERIFY REFRESH TOKEN AND RETURN NEW ACCESS TOKEN */
func HandleRefreshAccessToken(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleRefreshAccessToken( )\n")

	/* PARSE AND VALIDATE REQUEST DATA */
	us := UserSession{}
	if err = c.BodyParser(&us); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	} // Json("HandleRefreshAccessToken(): -> c.BodyParser(&us) -> user session", us)

	err = us.RefreshAccessToken()
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("Login refresh failed: %s", err.Error()),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":       "success",
		"message":      fmt.Sprintf("Welcome back citizen!"),
		"user_session": us,
	})
}

func HandleLogoutUser(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleLogoutUser( )\n")

	/* PARSE AND VALIDATE REQUEST DATA */
	us := UserSession{}
	if err = c.BodyParser(&us); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}  // Json("HandleLogoutUser(): -> c.BodyParser(&us) -> user session", us)

	us.LogoutUser()

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"message": "You have logged out.",
	})
}

/* REVOKE ACCESS BASED ON A USER ID ( ALL SESSIONS ) */
func HandleTerminateUserSessions(c *fiber.Ctx) (err error) {
	// fmt.Printf("\nHandleTerminateUserSessions( )\n")

	/* CHECK USER PERMISSION */
	if !UserRole_Admin(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to revoke user access",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	ur := UserResponse{}
	if err = c.BodyParser(&ur); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}
	Json("HandleTerminateUserSessions( ) -> c.BodyParser( ) -> ur", ur)

	count := TerminateUserSessions(ur)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": fmt.Sprintf("%d user sessions terminated.", count),
	})
}


/********************************************************************************************************/
/* NOT IMPLEMENTED: INTENDED AS API ENDPOINT FOR D2D CORE  *******************************/

/* NOT IMPLEMENTED: INTENDED AS API ENDPOINT FOR D2D CORE  */
func HandleRegisterDESDevice(c *fiber.Ctx) (err error) {

	/* CHECK USER PERMISSION */
	if !UserRole_Admin(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to register DES devices",
		})
	}

	/* PARSE AND VALIDATE REQUEST DATA */
	device := DESDev{}
	if err = c.BodyParser(&device); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": err.Error(),
		})
	}

	/* TODO: VALIDATE SERIAL# ( TO UPPER ):
	!= ""
	DOESN'T ALREADY EXIST
	LENGTH < 10
	*/

	/*
		CREATE A DEVICE RECORD IN THE DES DB FOR THIS DEVICE
		 CREATE A JOB RECORD IN THE DES DB FOR THIS DEVICE CMDARCHIVE
	*/
	reg, err := RegisterDESDevice(c.IP(), device)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   fiber.Map{"device": &reg},
	})
}

/* NOT IMPLEMENTED: INTENDED AS API ENDPOINT FOR D2D CORE  */
func HandleGetDESDeviceList(c *fiber.Ctx) (err error) {

	/* CHECK USER PERMISSION */
	if !UserRole_Admin(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to view DES device list",
		})
	}

	des_devs, err := GetDESDeviceList()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": err,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"devices": des_devs},
	})
}

/* NOT IMPLEMENTED: INTENDED AS API ENDPOINT FOR D2D CORE  */
func HandleGetDESDeviceBySerial(c *fiber.Ctx) (err error) {

	/* CHECK USER PERMISSION */
	if !UserRole_Admin(c.Locals("role")) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "fail",
			"message": "You must be an administrator to view DES devices",
		})
	}

	reg := DESRegistration{}
	if err = c.BodyParser(&reg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("GetDesDevBySerial(...) -> BodyParser failed:\n%s\n", err.Error()),
		})
	}

	if res := DES.DB.Order("des_dev_reg_time desc").First(&reg.DESDev, "des_dev_serial =?", reg.DESDev.DESDevSerial); res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("GetDesDevBySerial(...) -> query failed:\n%s\n", res.Error.Error()),
			"data":    fiber.Map{"device": reg},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "You are a tolerable person!",
		"data":    fiber.Map{"device": reg},
	})
}
