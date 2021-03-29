package backend

import (
	"encoding/json"
	"net/http"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"gopkg.in/gorp.v2"
)

func userOwnsFamily(db *gorp.DbMap, user_id int64, family_id int64) bool {
	result, err := db.Get(Family{}, family_id)
	if err != nil {
		return false
	}

	family, ok := result.(*Family)
	if !ok {
		return false
	}

	return (family.UserId == user_id)
}

func ListFamilies(session sessions.Session, db *gorp.DbMap, params martini.Params, req *http.Request, ren render.Render) {
	userId := session.Get("UserId")
    if userId == nil {
        ren.JSON(http.StatusUnauthorized, nil)
        return
    }

	families := []Family{}

	_, err := db.Select(&families, "SELECT families.* FROM families JOIN familymembers ON families.id=familymembers.family_id WHERE familymembers.user_id=?", userId)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	ren.JSON(http.StatusOK, families)
}

func GetFamily(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	result, err := db.Get(Family{}, familyId)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	family, ok := result.(*Family)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, family)
}

func UpdateFamily(session sessions.Session, db *gorp.DbMap, params martini.Params, req *http.Request, ren render.Render) {
	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	userId := session.Get("UserId")
	if userId == nil {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	result, err := db.Get(Family{}, familyId)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	family, ok := result.(*Family)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	if family.UserId != userId.(int64) {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&family)
	if err != nil {
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	// No messing around with protected fields.
	family.UserId = userId.(int64)

	_, err = db.Update(family)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, family)
}
