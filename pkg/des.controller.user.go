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
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"  // go get github.com/golang-jwt/jwt
	"github.com/google/uuid"     // go get github.com/google/uuid
	"golang.org/x/crypto/bcrypt" // go get golang.org/x/crypto/bcrypt

	"github.com/gofiber/websocket/v2"
)

/* https://codevoweb.com/how-to-properly-use-jwt-for-authentication-in-golang/ */

const USER_SESSION_WS_KEEP_ALIVE_SEC = 30

type UserSession struct {
	SID       uuid.UUID     `json:"sid"`
	REFTok    string        `json:"ref_token"`
	ACCTok    string        `json:"acc_token"`
	USR       UserResponse  `json:"user"`
	DataOut   chan string   `json:"-"`
	Close     chan struct{} `json:"-"`
	CloseSend chan struct{} `json:"-"`
	CloseKeep chan struct{} `json:"-"`
}

type UserSessionMap map[string]UserSession

var UserSessionsMap = make(UserSessionMap)
var UserSessionsMapRWMutex = sync.RWMutex{}

func UserSessionsMapWrite(u UserSession) (err error) {

	sid := u.SID.String()
	if !ValidateUUIDString(sid) {
		err = fmt.Errorf("Invalid user session ID.")
		return
	}

	UserSessionsMapRWMutex.Lock()
	UserSessionsMap[sid] = u
	UserSessionsMapRWMutex.Unlock()
	return
}
func UserSessionsMapRead(sid string) (u UserSession, err error) {
	UserSessionsMapRWMutex.Lock()
	u = UserSessionsMap[sid]
	UserSessionsMapRWMutex.Unlock()

	if !ValidateUUIDString(u.SID.String()) {
		err = fmt.Errorf("User session not found; please log in.")
	}
	return
}
func UserSessionsMapCopy() (usm UserSessionMap) {
	UserSessionsMapRWMutex.Lock()
	usm = UserSessionsMap
	UserSessionsMapRWMutex.Unlock()
	return
}
func UserSessionsMapRemove(usid string) {
	UserSessionsMapRWMutex.Lock()
	delete(UserSessionsMap, usid)
	UserSessionsMapRWMutex.Unlock()
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

func GetUserByID(uid string) (user User, err error) {

	DES.DB.First(&user, "id = ?", uid)
	if user.ID.String() != uid {
		err = fmt.Errorf(ERR_AUTH_USER_NOT_FOUND)
	}
	return
}

func GetUserReferenceSRC(uid string) (src DESMessageSource, err error) {
	//ERR_USER_NOT_FOUND
	user, err := GetUserByID(uid)
	if err != nil {
		return
	}

	src.Time = time.Now().UTC().UnixMilli()
	src.Addr = user.Email
	src.UserID = user.ID.String()
	src.App = DES_APP
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
		return fmt.Errorf("Authorization failed. Your refresh token has expired; please log in.")
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

/* UserSession WEBSOCKET CONNECTION *** DO NOT RUN IN GO ROUTINE *** */
func (us *UserSession) UserSessionWS_Connect(ws *websocket.Conn) {

	start := time.Now().Unix()

	us.DataOut = make(chan string)
	us.Close = make(chan struct{})
	us.CloseSend = make(chan struct{})
	us.CloseKeep = make(chan struct{})

	/* LISTEN FOR MESSAGES FROM CONNECTED USER */
	go us.ListenForMessages(ws, start)

	/* KEEP ALIVE GO ROUTINE SEND "live" EVERY 30 SECONDS TO PREVENT WS DISCONNECT */
	go us.RunKeepAlive(start)

	/* SEND MESSAGES TO CONNECTED USER */
	go us.SendMessages(ws, start)

	UserSessionsMapWrite(*us)

	// fmt.Printf("\n(*UserSession) UserSessionWS_Connect() -> %s : %d -> OPEN.\n", us.USR.Name, start)
	open := true
	for open {
		select {
		case <-us.Close:
			if us.CloseKeep != nil {
				us.CloseKeep = nil
			}
			if us.CloseSend != nil {
				us.CloseSend = nil
			}
			if us.DataOut != nil {
				us.DataOut = nil
			}
			open = false
		}
	}
	// fmt.Printf("\n(*UserSession) UserSessionWS_Connect() -> %s : %d -> CLOSED.\n", us.USR.Name, start)
}

/* GO ROUTINE: LISTEN FOR MESSAGES FROM CONNECTED USER */
func (us *UserSession) ListenForMessages(ws *websocket.Conn, start int64) {
	listen := true
	for listen {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			// fmt.Printf("\n(*UserSession) ListenForMessages() %s -> ERROR: %s\n", us.USR.Email, err.Error())
			if strings.Contains(err.Error(), "close") {
				msg = []byte("close")
			}
		}
		/* CHECK IF USER HAS CLOSED THE CONNECTION */
		if string(msg) == "close" {
			us.CloseKeep <- struct{}{}
			us.CloseSend <- struct{}{}
			listen = false
		}
	} // fmt.Printf("\n(*UserSession) ListenForMessages() -> %s : %d -> DONE.\n", us.USR.Name, start)
	us.Close <- struct{}{}
}

/* GO ROUTINE: SEND WSMessage PERIODICALLY TO PREVENT WS DISCONNECT */
func (us *UserSession) RunKeepAlive(start int64) {
	msg := fmt.Sprintf("%s : %d", us.USR.Name, start)
	count := 0
	live := true
	for live {
		select {

		case <-us.CloseKeep:
			live = false

		default:
			if count == USER_SESSION_WS_KEEP_ALIVE_SEC {
				js, err := json.Marshal(&WSMessage{Type: "live", Data: msg})
				if err != nil {
					LogErr(err)
				}
				us.DataOut <- string(js)
				count = 0
			}
			time.Sleep(time.Second * 1)
			count++
		}
	} // fmt.Printf("\n(*UserSession) RunKeepAlive() -> %s : %d -> DONE.\n", us.USR.Name, start)
}

/* GO ROUTINE: SEND MESSAGES TO CONNECTED USER */
func (us *UserSession) SendMessages(ws *websocket.Conn, start int64) {
	send := true
	for send {
		select {

		case <-us.CloseSend:
			send = false

		case data := <-us.DataOut:
			if err := ws.WriteJSON(data); err != nil {
				if !strings.Contains(err.Error(), "close sent") {
					LogErr(err)
				}
			}
		}
	} // fmt.Printf("\n(*UserSession) SendMessages -> %s : %d -> DONE.\n", us.USR.Name, start)
}
