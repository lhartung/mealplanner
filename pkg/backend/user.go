package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gorp.v2"
)

type changePasswordRequest struct {
	Current  string `json:"current"`
	Password string `json:"password"`
}

type changePasswordResponse struct {
	Message string `json:"message"`
}

func GetUser(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	id, err := strconv.ParseInt(params["id"], 0, 64)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	admin := session.Get("Admin")
	userId := session.Get("UserId")
	if userId == nil {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	if !(id == userId.(int64) || admin.(bool)) {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	result, err := db.Get(User{}, id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	user, ok := result.(*User)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	// Do not return the user's password.
	user.Password = ""

	ren.JSON(http.StatusOK, user)
}

func UpdateUser(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	id, err := strconv.ParseInt(params["id"], 0, 64)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	admin := session.Get("Admin")
	userId := session.Get("UserId")
	if userId == nil {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	if !(id == userId.(int64) || admin.(bool)) {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	result, err := db.Get(User{}, id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	user, ok := result.(*User)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	newUser := *user
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&newUser)
	if err != nil {
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	// Check if user already exists.
	count, _ := db.SelectInt("SELECT COUNT(email) FROM users WHERE id!=? AND email=?", userId, newUser.Email)
	if count > 0 {
		ren.JSON(http.StatusConflict, nil)
		return
	}

	if newUser.Email != user.Email {
		user.EmailVerified.SetValid(false)
	}

	// We only allow updating certain fields.
	user.Email = newUser.Email
	user.Name = newUser.Name
	user.UserName = newUser.Email

	_, err = db.Update(user)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, user)
}

func UpdateUserPassword(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	response := changePasswordResponse{}

	id, err := strconv.ParseInt(params["id"], 0, 64)
	if err != nil {
		logError(err)
		response.Message = "Error reading user ID"
		ren.JSON(http.StatusBadRequest, response)
		return
	}

	admin := session.Get("Admin")
	userId := session.Get("UserId")
	if userId == nil {
		response.Message = "Unauthorized"
		ren.JSON(http.StatusUnauthorized, response)
		return
	}

	if !(id == userId.(int64) || admin.(bool)) {
		response.Message = "Unauthorized"
		ren.JSON(http.StatusUnauthorized, response)
		return
	}

	result, err := db.Get(User{}, id)
	if err != nil {
		logError(err)
		response.Message = "Error reading user from database."
		ren.JSON(http.StatusNotFound, response)
		return
	}

	user, ok := result.(*User)
	if !ok {
		response.Message = "Error loading user"
		ren.JSON(http.StatusInternalServerError, response)
		return
	}

	request := changePasswordRequest{}
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&request)
	if err != nil {
		response.Message = "Error parsing request"
		ren.JSON(http.StatusBadRequest, response)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Current))
	if err != nil {
		logError(err)
		response.Message = "Current password is incorrect."
		ren.JSON(http.StatusBadRequest, response)
		return
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(request.Password), -1)
	user.Password = string(hashed)

	_, err = db.Update(user)
	if err != nil {
		logError(err)
		response.Message = "Error saving user"
		ren.JSON(http.StatusInternalServerError, response)
		return
	}

	response.Message = "Password changed."
	ren.JSON(http.StatusOK, response)
}

func SendEmailVerification(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	id, err := strconv.ParseInt(params["id"], 0, 64)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	admin := session.Get("Admin")
	userId := session.Get("UserId")
	if userId == nil {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	if !(id == userId.(int64) || admin.(bool)) {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	result, err := db.Get(User{}, id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	user, ok := result.(*User)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	url := fmt.Sprintf("https://example.com/user/%d/verify.html?token=%s", user.Id, user.EmailToken)

	htmlBody := fmt.Sprintf(
		"<h1>Meal Planner</h1>"+
			"<p>Please verify your email address by clicking the link below.</p>"+
			"<p><a href='%s'>%s</a></p>",
		url, url)

	textBody := "Meal Planner\r\n" +
		"Please verify your email address by opening the link below in a web browser.\r\n" +
		url + "\r\n"

	err = SendEmail(user.Email, "Meal Planner - Email Verification", htmlBody, textBody)
	if err != nil {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, user)
}
