package backend

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"gopkg.in/gorp.v2"
)

func ListMembers(session sessions.Session, db *gorp.DbMap, params martini.Params, req *http.Request, ren render.Render) {
	members := []FamilyMember{}

	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	_, err := db.Select(&members, "SELECT * FROM familymembers WHERE family_id=?", familyId)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	ren.JSON(http.StatusOK, members)
}

func CreateMember(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	member := FamilyMember{}

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&member)
	if err != nil {
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	// No messing around with protected fields.
	member.FamilyId = familyId

	err = db.Insert(&member)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, member)
}

func DeleteMember(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		logError(err)
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	result, err := db.Get(FamilyMember{}, id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	member, ok := result.(*FamilyMember)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

    result, err = db.Get(Family{}, familyId)
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

	// Family owner cannot be removed.
	if member.UserId == family.UserId {
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	_, err = db.Delete(member)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, member)
}
