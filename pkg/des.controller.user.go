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

	"github.com/golang-jwt/jwt"   // go get github.com/golang-jwt/jwt
	"github.com/google/uuid"      // go get github.com/google/uuid
	"golang.org/x/crypto/bcrypt"  // go get golang.org/x/crypto/bcrypt
)

/* https://codevoweb.com/how-to-properly-use-jwt-for-authentication-in-golang/ */

type UserSession struct {
	SID    uuid.UUID    `json:"sid"`
	REFTok string       `json:"ref_token"`
	ACCTok string       `json:"acc_token"`
	USR    UserResponse `json:"user"`
}

type UserSessionMap map[string]UserSession

var UserSessions = make(UserSessionMap)
var UserSessionsRWMutex = sync.RWMutex{}

func UserSessionsMapWrite(u UserSession) (err error) {

	sid := u.SID.String()
	if sid == "" || sid == "00000000-0000-0000-0000-000000000000" {
		err = fmt.Errorf("Invalid user session ID.")
		return
	}

	UserSessionsRWMutex.Lock()
	UserSessions[sid] = u
	UserSessionsRWMutex.Unlock()
	return
}
func UserSessionsMapRead(sid string) (u UserSession, err error) {
	UserSessionsRWMutex.Lock()
	u = UserSessions[sid]
	UserSessionsRWMutex.Unlock()

	if u.SID.String() == "00000000-0000-0000-0000-000000000000" {
		err = fmt.Errorf("User session not found. Please log in.")
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

/* USED TO REFRESH ACCESS TOKENS */
func (us *UserSession) UpdateMappedAccTok() (err error) {
	u, err := UserSessionsMapRead(us.SID.String())
	if err != nil {
		return
	}
	u.ACCTok = us.ACCTok
	err = UserSessionsMapWrite(u)
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

/* CREATE A NEW USER WITH DEFAULT ROLES */
func RegisterUser(runp RegisterUserInput) (user User, err error) {

	pwHash, err := bcrypt.GenerateFromPassword([]byte(runp.Password), bcrypt.DefaultCost)
	if err != nil {
		err = fmt.Errorf("Failed to hash password: %s", err.Error())
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
			err = fmt.Errorf("Failed to create user in database: %s", res.Error.Error())
		}
	}

	return
}

/* AUTHENTICATE USER INPUT AND RETURN JWTs */
func LoginUser(lunp LoginUserInput) (us UserSession, err error) {

	user := User{}
	/* CHECK EMAIL */
	res := DES.DB.First(&user, "email = ?", strings.ToLower(lunp.Email))
	if res.Error != nil {
		err = fmt.Errorf("Invalid email or password")
		return
	} // Json("LoginUser() -> user:", user)

	/* CHECK PASSWORD */
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(lunp.Password)); err != nil {
		err = fmt.Errorf("Invalid email or password")
		return
	}

	/* CREATE A USER SESSION ID */
	us.SID = uuid.New()

	/*  FILTER USER DATA */
	us.USR = user.FilterUserRecord() // Json("LoginUser() -> user session:", us)

	/* CREATE REFRESH TOKEN*/
	err = us.CreateJWTRefreshToken(JWT_REFRESH_EXPIRED_IN)
	if err != nil {
		err = fmt.Errorf("Refresh token generation failed: %s", err.Error())
		return
	}

	/* CREATE ACCESS TOKEN */
	err = us.CreateJWTAccessToken()
	if err != nil {
		err = fmt.Errorf("Access token generation failed: %s", err.Error())
		return
	}

	/* UPDATE USER SESSION MAP */
	err = UserSessionsMapWrite(us)

	return
}

/* REMOVES ALL SESSIONS FOR GIVEN USER FROM UserSessionsMap */
func TerminateUserSessions(ur UserResponse) (count int) {

	sess := UserSessionsMapCopy()

	count = 0
	for sid, us := range sess {
		if us.USR.ID == ur.ID {
			UserSessionsMapRemove(sid)
			count++
		}
	}

	return
}

/* RETURNS ALL TOKEN CLAIMS */
func GetClaimsFromTokenString(token string) (claims jwt.MapClaims, err error) {

	/* PARSE TOKEN STRING */
	tokenByte, err := jwt.Parse(token, func(jwtToken *jwt.Token) (interface{}, error) {
		if _, jwt_err := jwtToken.Method.(*jwt.SigningMethodHMAC); !jwt_err {
			return nil, fmt.Errorf("Unexpected signing method: %s", jwtToken.Header["alg"])
		}
		return []byte(JWT_SECRET), nil
	})
	if err != nil {
		return
	}

	/* GET THE USER ROLE & PASS ALONG TO THE NEXT HANDLER */
	claims, ok := tokenByte.Claims.(jwt.MapClaims)
	if !ok || !tokenByte.Valid {
		err = fmt.Errorf("Invalid token claim.")
		return
	}
	return
}

/* REMOVES THE SESSION FOR GIVEN USER FROM UserSessionsMap */
func (us *UserSession) LogoutUser() {
	UserSessionsMapRemove(us.SID.String())
}

/* CREATES A NEW ACCESS TOKEN IF REFRESH TOKEN HAS NOT EXPIRED */
func (us *UserSession) RefreshAccessToken() (err error) {
	// fmt.Printf("\nRefreshAccessToken( )")

	/* GET USER FROM SESSION MAP */
	mus, err := UserSessionsMapRead(string(us.SID.String()))
	if err != nil {
		return
	} // Json("RefreshAccessToken( ) -> UserSessionsMapRead( ) -> mus: ", mus)

	/* CHECK REFRESH TOKEN EXPIRE DATE IN MAPPED USER SESSION. IF TIMEOUT, DENY */
	ref_claims, err := GetClaimsFromTokenString(mus.REFTok)
	if err != nil {
		return err
	}
	exp := 0
	now := int(time.Now().Unix())
	if fExp, ok := ref_claims["exp"].(float64); ok {
		exp = int(fExp)
	} // fmt.Printf("\nRefreshAccessToken( ) -> exp: %d", exp) // fmt.Printf("\nRefreshAccessToken( ) -> now: %d\n", now)

	if exp < now {
		return fmt.Errorf("Your refresh token has expired. Please log in.")
	}

	if err = us.CreateJWTAccessToken(); err != nil {
		return
	}

	return UserSessionsMapWrite(*us)
}

/* CREATES A JWT REFRESH TOKEN; USED ON LOGIN ONLY */
func (us *UserSession) CreateJWTRefreshToken(dur time.Duration) (err error) {

	tokByte := jwt.New(jwt.SigningMethodHS256)
	tokClaims := tokByte.Claims.(jwt.MapClaims)
	tokClaims["sub"] = us.USR.ID // SUBJECT
	tokClaims["exp"] = time.Now().UTC().Add(dur).Unix()

	us.REFTok, err = tokByte.SignedString([]byte(JWT_SECRET))
	if err != nil {
		err = fmt.Errorf("Failed to sign refresh token: %s", err.Error())
	}
	return
}

/* CREATES A JWT ACCESS TOKEN; USED ON LOGIN AND SUBSEQUENT REFRESHES */
func (us *UserSession) CreateJWTAccessToken() (err error) {

	now := time.Now().UTC()

	/* TODO: BUILD MORE COMPLEX CLAIMS OBJECT WITH:
	DES ROLE
	DEVICE-SPECIFIC ROLES
	JOB-SPECIFIC ROLES
	*/

	/* CREATE JWT CLAIMS FOR A GIVEN USER */
	claims := jwt.MapClaims{
		"sub": us.USR.ID,   // SUBJECT
		"rol": us.USR.Role, // ROLE
		"exp": now.Add(JWT_EXPIRED_IN).Unix(),
		"iat": now.Unix(), // ISSUED AT
		"nbf": now.Unix(), // NOT VALID BEFORE
	}
	tokenByte := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	us.ACCTok, err = tokenByte.SignedString([]byte(JWT_SECRET))
	if err != nil {
		err = fmt.Errorf("Failed to sign access token: %s", err.Error())
	}
	return
}

func GetUserByID(userID interface{}) (user User, err error) {

	DES.DB.First(&user, "id = ?", userID)
	if user.ID.String() != userID {
		err = fmt.Errorf("The user belonging to this token no logger exists.")
	}
	return
}

func GetUserList() (users []UserResponse, err error) {

	qry := DES.DB.Table("users").Select("*")

	us := []User{}
	res := qry.Scan(&us)
	if res.Error != nil {
		err = fmt.Errorf("Failed to retrieve users from database: %s", res.Error.Error())
		return
	}

	for _, user := range us {
		users = append(users, user.FilterUserRecord())
	}

	return
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
