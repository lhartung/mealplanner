package backend

import (
	"net/http"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gorp.v2"
)

func GetToken(session sessions.Session, db *gorp.DbMap, params martini.Params, req *http.Request, ren render.Render) {
	response := AuthResponse{}

	email := req.FormValue("email")
	password := req.FormValue("password")

	user := User{}
	err := db.SelectOne(&user,
		"SELECT * FROM users WHERE email=? LIMIT 1", email)
	if err != nil {
		logError(err)
		response.Message = "Email address was not recognized."
		ren.JSON(http.StatusBadRequest, &response)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		logError(err)
		response.Message = "Password is incorrect."
		ren.JSON(http.StatusBadRequest, &response)
		return
	}

	memberships := []FamilyMember{}
	_, err = db.Select(&memberships, "SELECT * FROM familymembers WHERE familymembers.user_id=? ORDER BY familymembers.family_id", user.Id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	families := make([]int64, 0)
	for _, membership := range memberships {
		families = append(families, membership.FamilyId)
	}

	session.Set("Admin", user.Admin.Bool)
	session.Set("Families", families)
	session.Set("FamilyId", user.DefaultFamilyId)
	session.Set("UserId", user.Id)

	//response.Token = tokenString
	response.UserId = user.Id
	response.Families = families

	ren.JSON(http.StatusOK, &response)
}

func ClearToken(session sessions.Session, db *gorp.DbMap, params martini.Params, req *http.Request, res http.ResponseWriter) {
	session.Clear()
	res.WriteHeader(http.StatusNoContent)
}

func CreateAccount(session sessions.Session, db *gorp.DbMap, params martini.Params, req *http.Request, ren render.Render) {
	response := AuthResponse{}

	username := req.FormValue("username")
	password := req.FormValue("password")
	email := req.FormValue("email")
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), -1)

	// Check if user already exists.
	count, _ := db.SelectInt("SELECT COUNT(email) FROM users WHERE email=?", email)
	if count > 0 {
		response.Message = "A user with that name or email address already exists."
		ren.JSON(http.StatusBadRequest, &response)
		return
	}

	user := User{
		UserName: email,
		Password: string(hashed),
		Email:    email,
		Name:     username,
	}

	err := db.Insert(&user)
	if err != nil {
		logError(err)
		response.Message = "Server error - please try again later."
		ren.JSON(http.StatusInternalServerError, &response)
		return
	}

	today := time.Now()
	family := Family{
		UserId:          user.Id,
		Name:            user.Name,
		CreatedOn:       today.Format(dateFormat),
		AccountStatus:   "trial",
		StatusExpiresOn: today.AddDate(0, 0, 30).Format(dateFormat),
	}

	err = db.Insert(&family)
	if err != nil {
		logError(err)
		response.Message = "Server error - please try again later."
		ren.JSON(http.StatusInternalServerError, &response)
		return
	}

	member := FamilyMember{
		FamilyId: family.Id,
		UserId:   user.Id,
		CanEdit:  true,
	}

	err = db.Insert(&member)
	if err != nil {
		logError(err)
		response.Message = "Server error - please try again later."
		ren.JSON(http.StatusInternalServerError, &response)
		return
	}

	user.DefaultFamilyId = family.Id

	_, err = db.Update(&user)
	if err != nil {
		logError(err)
		response.Message = "Server error - please try again later."
		ren.JSON(http.StatusInternalServerError, &response)
		return
	}

	families := []int64{family.Id}

	//response.Token = tokenString
	response.UserId = user.Id
	response.Families = families

	session.Set("Admin", user.Admin.Bool)
	session.Set("Families", families)
	session.Set("FamilyId", user.DefaultFamilyId)
	session.Set("UserId", user.Id)

	ren.JSON(http.StatusOK, &response)
}
