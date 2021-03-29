package backend

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"gopkg.in/gorp.v2"
	"gopkg.in/guregu/null.v3"
)

type AssignedIngredientView struct {
	// Assignment table
	AssignmentId int64  `db:"id" json:"assignment_id"`
	RecipeId     int64  `db:"recipe_id" json:"recipe_id"`
	Date         string `db:"date" json:"date"`
	Meal         string `db:"meal" json:"meal"`

	// Ingredient table
	IngredientId int64       `db:"ingredient_id" json:"ingredient_id"`
	ClassId      null.Int    `db:"class_id" json:"class_id"`
	Name         string      `db:"name" json:"name"`
	Amount       null.String `db:"amount" json:"amount"`
	Have         null.Bool   `db:"have" json:"have"`

	// IngredientClass table
	Class null.String `db:"class" json:"class"`
}

// Extended Ingredient structure to accept update requests.
type ExtendedIngredient struct {
	Ingredient

	Class null.String `json:"class"`
}

func (item *AssignedIngredientView) setClassLocal(db *gorp.DbMap, id int64) {
	result, err := db.Get(IngredientClass{}, id)
	if err != nil || result == nil {
		item.ClassId.Int64 = 0
		item.ClassId.Valid = false
		item.Class.String = ""
		item.Class.Valid = false
	} else {
		ingclass := (result).(*IngredientClass)
		item.ClassId.Int64 = id
		item.ClassId.Valid = true
		item.Class.String = ingclass.Name
		item.Class.Valid = true
	}
}

func ListIngredients(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	var err error
	query := req.URL.Query()
	ingredients := []Ingredient{}

	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	recipe_id, err := strconv.Atoi(query.Get("recipe_id"))
	if err == nil {
		_, err = db.Select(&ingredients,
			"SELECT * FROM ingredients WHERE owner_id=? AND recipe_id=?",
			familyId, recipe_id)
	} else {
		_, err = db.Select(&ingredients,
			"SELECT * FROM ingredients WHERE owner_id=?", familyId)
	}

	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	ren.JSON(http.StatusOK, ingredients)
}

func CreateIngredient(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	ingredient := Ingredient{}

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&ingredient)
	if err != nil {
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	// No messing around with protected fields.
	ingredient.OwnerId = familyId

	// If passing the class by name, we will have to look up the corresponding
	// class id.
	class := req.FormValue("class")
	if class != "" {
		ingclass := IngredientClass{}
		err := db.SelectOne(&ingclass,
			"SELECT * FROM ingclasses WHERE name=?", class)
		if err != nil {
			logError(err)
			ren.JSON(http.StatusBadRequest, nil)
			return
		}

		ingredient.ClassId.Int64 = ingclass.Id
		ingredient.ClassId.Valid = true
	}

	err = db.Insert(&ingredient)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, ingredient)
}

func ListAssignedIngredients(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	query := req.URL.Query()
	var err error

	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	fromDate := query.Get("from")
	if fromDate == "" {
		fromDate = time.Now().Format("2006-01-02")
	}

	toDate := query.Get("to")
	if toDate == "" {
		t := time.Now().AddDate(0, 0, 30)
		toDate = t.Format("2006-01-02")
	}

	assignments := []AssignedIngredientView{}
	_, err = db.Select(&assignments,
		"SELECT a.id, a.recipe_id, a.date, a.meal, "+
			"i.id AS ingredient_id, i.class_id, i.name, i.amount, i.have, "+
			"c.name AS class "+
			"FROM assignments a, ingredients i "+
			"LEFT OUTER JOIN ingclasses c ON (c.id=i.class_id) "+
			"WHERE a.date>=? AND a.date<? AND a.recipe_id=i.recipe_id AND a.owner_id=? "+
			"ORDER BY a.date",
		fromDate, toDate, familyId)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	for i, item := range assignments {
		if !item.ClassId.Valid {
			class_id := Classifier.Classify(item.Name, item.Amount.String)
			item.setClassLocal(db, class_id)
			assignments[i] = item
		}
	}

	ren.JSON(http.StatusOK, assignments)
}

func GetIngredient(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
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

	result, err := db.Get(Ingredient{}, id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	} else if result == nil {
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	ingredient, ok := result.(*Ingredient)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	if ingredient.OwnerId != familyId {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	ren.JSON(http.StatusOK, ingredient)
}

func UpdateIngredient(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
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

	result, err := db.Get(Ingredient{}, id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	orig_ingredient, ok := result.(*Ingredient)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	// Convert up to ExtendedIngredient, which has class string.
	ingredient := ExtendedIngredient{
		Ingredient: *orig_ingredient,
	}

	if ingredient.OwnerId != familyId {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&ingredient)
	if err != nil {
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	// No messing around with protected fields.
	ingredient.OwnerId = familyId

	// If passing the class by name, we will have to look up the corresponding
	// class id.
	if ingredient.Class.Valid {
		ingclass := IngredientClass{}
		err = db.SelectOne(&ingclass,
			"SELECT * FROM ingclasses WHERE name=?", ingredient.Class.String)
		if err != nil {
			logError(err)
			ren.JSON(http.StatusBadRequest, nil)
			return
		}

		ingredient.ClassId.Int64 = ingclass.Id
		ingredient.ClassId.Valid = true
	}

	_, err = db.Update(&ingredient.Ingredient)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, ingredient)
}

func DeleteIngredient(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
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

	result, err := db.Get(Ingredient{}, id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	} else if result == nil {
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	ingredient, ok := result.(*Ingredient)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	if ingredient.OwnerId != familyId {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	_, err = db.Delete(ingredient)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, ingredient)
}
