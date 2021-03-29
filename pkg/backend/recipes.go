package backend

import (
	"bufio"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"gopkg.in/gorp.v2"
	"gopkg.in/guregu/null.v3"
)

type AssignedRecipeView struct {
	// Assignment table
	AssignmentId int64  `db:"id" json:"assignment_id"`
	RecipeId     int64  `db:"recipe_id" json:"recipe_id"`
	Date         string `db:"date" json:"date"`
	Meal         string `db:"meal" json:"meal"`

	// Recipe table
	Fixed  bool        `db:"fixed" json:"fixed"`
	Name   string      `db:"name" json:"name"`
	Source null.String `db:"source" json:"source"`
}

func ListRecipes(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	recipes := []Recipe{}

	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	filter := req.URL.Query().Get("filter")
	var err error
	if filter == "available" {
		_, err = db.Select(&recipes, "SELECT * FROM recipes WHERE show=1 AND owner_id=?", familyId)
	} else {
		_, err = db.Select(&recipes, "SELECT * FROM recipes WHERE owner_id=?", familyId)
	}

	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	ren.JSON(http.StatusOK, recipes)
}

func CreateRecipe(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	recipe := Recipe{
		Show:  true,
		Fixed: false,
	}

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&recipe)
	if err != nil {
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	// No messing around with protected fields.
	recipe.OwnerId = familyId

	err = db.Insert(&recipe)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, recipe)
}

func ListAssignedRecipes(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	query := req.URL.Query()

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

	recipes := []AssignedRecipeView{}
	_, err := db.Select(&recipes,
		"SELECT a.id, a.recipe_id, a.date, a.meal, "+
			"r.fixed, r.name, r.source "+
			"FROM assignments a, recipes r "+
			"WHERE a.recipe_id=r.id AND a.date>=? AND a.date<? AND a.owner_id=?",
		fromDate, toDate, familyId)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	ren.JSON(http.StatusOK, recipes)
}

func GetRecipe(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
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

	result, err := db.Get(Recipe{}, id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	} else if result == nil {
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	recipe, ok := result.(*Recipe)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	if recipe.OwnerId != familyId {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	ren.JSON(http.StatusOK, recipe)
}

func UpdateRecipe(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
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

	result, err := db.Get(Recipe{}, id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	recipe, ok := result.(*Recipe)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	if recipe.OwnerId != familyId {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&recipe)
	if err != nil {
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	// No messing around with protected fields.
	recipe.OwnerId = familyId

	_, err = db.Update(recipe)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, nil)
}

func DeleteRecipe(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
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

	result, err := db.Get(Recipe{}, id)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusNotFound, nil)
		return
	}

	recipe, ok := result.(*Recipe)
	if !ok {
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	if recipe.OwnerId != familyId {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	_, err = db.Delete(recipe)
	if err != nil {
		logError(err)
		ren.JSON(http.StatusInternalServerError, nil)
		return
	}

	ren.JSON(http.StatusOK, recipe)
}

func ImportRecipes(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, ren render.Render) {
	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		ren.JSON(http.StatusUnauthorized, nil)
		return
	}

	file, _, err := req.FormFile("file")
	defer file.Close()

	if err != nil {
		logError(err)
		ren.JSON(http.StatusBadRequest, nil)
		return
	}

	addedRecipes := []Recipe{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		recipe := Recipe{
			OwnerId: familyId,
			Show:    true,
			Fixed:   false,
		}

		parts := strings.Split(scanner.Text(), ",")
		if len(parts) >= 1 {
			recipe.Name = parts[0]
		}
		if len(parts) >= 2 {
			recipe.Source.String = parts[1]
			recipe.Source.Valid = true
		}

		err = db.Insert(&recipe)
		if err != nil {
			logError(err)
			ren.JSON(http.StatusInternalServerError, nil)
			return
		}

		if len(parts) >= 3 {
			pattern, _ := regexp.Compile("(.*)[[:space:]]*\\((.*)\\)")

			ingredients := strings.Split(parts[2], ":")
			for _, ing := range ingredients {
				ingredient := Ingredient{
					OwnerId: familyId,
				}

				ingredient.RecipeId.Int64 = recipe.Id
				ingredient.RecipeId.Valid = true

				substrings := pattern.FindStringSubmatch(ing)
				if len(substrings) >= 3 {
					ingredient.Name = substrings[1]
					ingredient.Amount.String = substrings[2]
					ingredient.Amount.Valid = true
				} else {
					ingredient.Name = ing
				}

				err = db.Insert(&ingredient)
				if err != nil {
					logError(err)
					ren.JSON(http.StatusInternalServerError, nil)
					return
				}
			}
		}

		addedRecipes = append(addedRecipes, recipe)
	}

	ren.JSON(http.StatusOK, addedRecipes)
}
