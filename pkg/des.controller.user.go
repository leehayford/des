/* Data Exchange Server (DES) is a component of the Datacan Data2Desk (D2D) Platform.
License:

	[PROPER LEGALESE HERE...]

	INTERIM LICENSE DESCRIPTION:
	In spirit, this license:
	1. Allows <Third Party> to use, modify, and / or distributre this software in perpetuity so long as <Third Party> understands:
		a. The software is porvided as is without guarantee of additional support from DataCan in any form.
		b. The software is porvided as is without guarantee of exclusivity.

	2. Prohibits <Third Party> from taking any action which might interfere with DataCan's right to use, modify and / or distributre this software in perpetuity.
*/

package pkg

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2" // go get github.com/gofiber/fiber/v2
	"github.com/golang-jwt/jwt"   // go get github.com/golang-jwt/jwt
	"golang.org/x/crypto/bcrypt"  // go get golang.org/x/crypto/bcrypt
	"github.com/google/uuid"                 // go get github.com/google/uuid
)

/* https://codevoweb.com/how-to-properly-use-jwt-for-authentication-in-golang/ */

type UserSession struct {
	SID uuid.UUID `json:"sid"`
	REFTok string `json:"ref_token"`
	ACCTok string `json:"acc_token"`
	USR UserResponse `json:"user"`
}

type UserSessionMap map[string]UserSession
var UserSessions = make(UserSessionMap)
var UserSessionsRWMutex = sync.RWMutex{}


func UserSessionsMapWrite(usid string, u UserSession) {
	UserSessionsRWMutex.Lock()
	UserSessions[usid] = u
	UserSessionsRWMutex.Unlock()
}
func UserSessionsMapRead(usid string) (u UserSession, err error) {
	UserSessionsRWMutex.Lock()
	u = UserSessions[usid]
	UserSessionsRWMutex.Unlock()
	
	if u.SID.String() == "" {
		err = fmt.Errorf("No session exists with ID %s.", usid)
	}
	return
}
func UserSessionsMapCopy() (usm UserSessionMap) {
	UserSessionsRWMutex.Lock()
	usm = UserSessions
	UserSessionsRWMutex.Unlock()
	return
}
func UserSessionsMapRemove(usid string) {
	UserSessionsRWMutex.Lock()
	delete(UserSessions, usid)
	UserSessionsRWMutex.Unlock()
}


func (us *UserSession) UpdateMappedAccTok() (err error) {
	u, err := UserSessionsMapRead(us.SID.String())
	if err != nil {
		return
	}
	u.ACCTok = us.ACCTok
	UserSessionsMapWrite(us.SID.String(), u)
	return
}
func (us *UserSession) UpdateMappedRefTok() (err error) {
	u, err := UserSessionsMapRead(us.SID.String())
	if err != nil {
		return
	}
	u.REFTok = us.REFTok
	UserSessionsMapWrite(us.SID.String(), u)
	return
}


func (us *UserSession) GetMappedAccTok() (err error) {
	u, err := UserSessionsMapRead(us.SID.String())
	if err != nil {
		return
	}
	us.ACCTok = u.ACCTok
	return
}
func (us *UserSession) GetMappedRefTok() (err error) {
	u, err := UserSessionsMapRead(us.SID.String())
	if err != nil {
		return
	}
	us.REFTok = u.REFTok
	return
}

func RefreshAccessToken(usid string) (acc string, err error) {
	fmt.Printf("\nRefreshAccessToken( ): \n%s\n", usid)


	us, err := UserSessionsMapRead(usid)
	if err != nil {
		return
	}

	// user, err := GetUserByID(us.USR.ID.String())
	user, err := GetUserByID(usid)
	if err != nil {
		return
	}

	acc, err = CreateJWTAccessToken(user)
	if err != nil {
		return
	}

	us.ACCTok = acc
	us.UpdateMappedAccTok()

	return
}

/* REMOVES ALL SESSIONS FOR GIVEN USER  */
func RevokeRefreshToken(user User) {

	sess := UserSessionsMapCopy()

	for id, ses := range sess {
		if ses.USR.ID == user.ID {
			UserSessionsMapRemove(id)
		}
	}
}

/* CREATE A NEW USER WITH DEFAULT ROLES */
func RegisterUser(runp RegisterUserInput) (user User, err error) {

	pwHash, err := bcrypt.GenerateFromPassword([]byte(runp.Password), bcrypt.DefaultCost)
	if err != nil {
		return
	}

	user = User{
		Name:     runp.Name,
		Email:    strings.ToLower(runp.Email),
		Password: string(pwHash),
		Role:     "user",
		Photo:    runp.Photo,
	}

	res := DES.DB.Create(&user)
	if res.Error != nil {
		if strings.Contains(res.Error.Error(), "duplicate key value violates unique") {
			err = fmt.Errorf("User with that email already exists")
		} else {
			err = res.Error
		}
	}

	return
}

/* AUTHENTICATE USER INPUT AND RETURN JWTs */
func LoginUser(lunp LoginUserInput) (ures UserResponse, acc, ref string, err error) {

	user := User{}
	/* CHECK EMAIL */
	res := DES.DB.First(&user, "email = ?", strings.ToLower(lunp.Email)) 
	if res.Error != nil {
		err = fmt.Errorf("Invalid email or password")
		return
	}

	/* CHECK PASSWORD */
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(lunp.Password)); err != nil {
		err = fmt.Errorf("Invalid email or password")
		return
	}

	/* CREATE REFRESH TOKEN*/
	ref, err = CreateJWTRefreshToken(user)
	if err != nil {
		err = fmt.Errorf("Refresh token generation failed: %v", err)
		return
	}

	/* CREATE ACCESS TOKEN */
	acc, err = CreateJWTAccessToken(user)
	if err != nil {
		err = fmt.Errorf("Access token generation failed: %v", err)
		return
	}

	/*  RETURN FILTERED USER DATA */
	ures = user.FilterUserRecord()

	return
}

func CreateJWTRefreshToken(user User) (tok string, err error) {

	tokByte := jwt.New(jwt.SigningMethodHS256)
	tokClaims := tokByte.Claims.(jwt.MapClaims)
	tokClaims["sub"] = user.ID // SUBJECT
	tokClaims["exp"] = time.Now().UTC().Add(time.Duration(time.Minute * 3)).Unix()

	tok, err = tokByte.SignedString([]byte(JWT_SECRET))
	if err != nil {
		return "", err
	}

	return
}

func CreateJWTAccessToken(user User) (acc string, err error) {

	claims, err := CreateJWTClaims(user)
	if err != nil {
		return "", err
	}

	tokenByte := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	acc, err = tokenByte.SignedString([]byte(JWT_SECRET))
	if err != nil {
		return "", err
	}

	return
}

/* CREATE JWT CLAIMS FOR A GIVEN USER */
func CreateJWTClaims(user User) (claims jwt.MapClaims, err error) {

	now := time.Now().UTC()

	/* TODO: BUILD MORE COMPLEX CLAIMS OBJECT WITH:
	DES ROLE
	DEVICE-SPECIFIC ROLES
	JOB-SPECIFIC ROLES
	*/

	claims = jwt.MapClaims{
		"sub": user.ID,   // SUBJECT
		"rol": user.Role, // ROLE
		"exp": now.Add(JWT_EXPIRED_IN).Unix(),
		"iat": now.Unix(), // ISSUED AT
		"nbf": now.Unix(), // NOT VALID BEFORE
	}

	return
}


func LogoutUser(c *fiber.Ctx) error {
	expired := time.Now().Add(-time.Hour * 24)
	c.Cookie(&fiber.Cookie{
		Name:    "token",
		Value:   "",
		Expires: expired,
	})
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
}

func GetUserByID(userID interface{}) (user User, err error) {

	DES.DB.First(&user, "id = ?", userID)
	if user.ID.String() != userID {
		err = fmt.Errorf("The user belonging to this token no logger exists.")
	}
	return
}

func GetUserList(c *fiber.Ctx) error {

	fmt.Printf("\nGetUsers( ):\n")

	users := []User{}
	DES.DB.Find(&users)
	fmt.Printf("\nusrs: %d\n", len(users))

	userList := []UserResponse{}
	for _, user := range users {
		userList = append(userList, user.FilterUserRecord())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "These are all tolerable people!",
		"data":    fiber.Map{"users": userList},
	})
}

func CreateDESUserForDevice(serial, pw string) (user UserResponse, err error) {

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	u := User{
		Name:     serial,
		Email:    fmt.Sprintf("%s@datacan.ca", strings.ToLower(serial)),
		Password: string(hashedPassword),
		Role:     "device",
	}
	result := DES.DB.Create(&u)
	err = result.Error
	user = u.FilterUserRecord()

	return
}
