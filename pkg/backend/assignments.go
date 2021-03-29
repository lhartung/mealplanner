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

func ListAssignments(session sessions.Session, db *gorp.DbMap, params martini.Params, req *http.Request, ren render.Render) {
	assignments := []Assignment{}
	query := req.URL.Query()
	var err error

	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	fromDate := query.Get("from")
	toDate := query.Get("to")

	if fromDate != "" && toDate != "" {
		_, err = db.Select(&assignments, "SELECT * FROM assignments WHERE date>=? AND date<=? AND owner_id=?", fromDate, toDate, familyId)
	} else {
		_, err = db.Select(&assignments, "SELECT * FROM assignments WHERE owner_id=?", familyId)
	}

	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	ren.JSON(http.StatusOK, assignments)
}

func CreateAssignment(session sessions.Session, db *gorp.DbMap, params martini.Params, req *http.Request, ren render.Render) {
	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	assignment := Assignment{}

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&assignment)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	// No messing around with protected fields.
	assignment.OwnerId = familyId

	err = db.Insert(&assignment)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, assignment)
}

func GetAssignment(session sessions.Session, db *gorp.DbMap, params martini.Params, req *http.Request, ren render.Render) {
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

	result, err := db.Get(Assignment{}, id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	} else if result == nil {
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	assignment, ok := result.(*Assignment)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	if assignment.OwnerId != familyId {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	ren.JSON(http.StatusOK, assignment)
}

func UpdateAssignment(session sessions.Session, db *gorp.DbMap, params martini.Params, req *http.Request, ren render.Render) {
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

	result, err := db.Get(Assignment{}, id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	assignment, ok := result.(*Assignment)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	if assignment.OwnerId != familyId {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&assignment)
	if err != nil {
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	// No messing around with protected fields.
	assignment.OwnerId = familyId

	_, err = db.Update(assignment)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, nil)
}

func DeleteAssignment(session sessions.Session, db *gorp.DbMap, params martini.Params, req *http.Request, ren render.Render) {
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

	result, err := db.Get(Assignment{}, id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	assignment, ok := result.(*Assignment)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	if assignment.OwnerId != familyId {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	_, err = db.Delete(assignment)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, assignment)
}
