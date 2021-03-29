package backend

import (
	"bufio"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/sessions"
	"gopkg.in/flosch/pongo2.v3"
	"gopkg.in/gorp.v2"
)

//var landing = pongo2.Must(pongo2.FromFile("templates/landing.html"))

const dateFormat = "2006-01-02"

func getAccountStatus(db *gorp.DbMap, session sessions.Session) (string, bool) {
	familyId := session.Get("FamilyId")
	if familyId == nil {
		return "", false
	}

	families := []Family{}
	_, err := db.Select(&families,
		"SELECT * FROM families WHERE id==?", familyId)
	if err != nil || len(families) == 0 {
		return "", false
	}

	today := time.Now()

	expiration, err := time.Parse(dateFormat, families[0].StatusExpiresOn)
	if err != nil {
		return families[0].AccountStatus, false
	} else {
		return families[0].AccountStatus, today.After(expiration)
	}
}

func getUser(db *gorp.DbMap, session sessions.Session) *User {
	userId := session.Get("UserId")
	if userId == nil {
		return nil
	}

	result, err := db.Get(User{}, userId)
	if err != nil {
		return nil
	}

	user, ok := result.(*User)
	if !ok {
		return nil
	}

	return user
}

func getFamilies(db *gorp.DbMap, session sessions.Session) []Family {
	families := []Family{}

	userId := session.Get("UserId")
	if userId == nil {
		return families
	}

	_, err := db.Select(&families,
		"SELECT f.* "+
			"FROM families f, familymembers m "+
			"WHERE m.user_id==? AND f.id==m.family_id",
		userId)
	if err != nil {
		logError(err)
	}

	return families
}

func ViewLandingPage(session sessions.Session, db *gorp.DbMap, params martini.Params, req *http.Request, res http.ResponseWriter) {
	user := getUser(db, session)

	context := pongo2.Context{
		"user": user,
	}

	landing, err := pongo2.FromCache("templates/landing.html")
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = landing.ExecuteWriter(context, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func ViewSignUpPage(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, res http.ResponseWriter) {
	user := getUser(db, session)

	context := pongo2.Context{
		"user": user,
	}

	landing, err := pongo2.FromCache("templates/sign-up.html")
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = landing.ExecuteWriter(context, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func ViewSignOutPage(db *gorp.DbMap, params martini.Params, req *http.Request, res http.ResponseWriter) {
	context := pongo2.Context{
		"user": nil,
	}

	landing, err := pongo2.FromCache("templates/landing.html")
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Set-Cookie", "auth=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT")

	err = landing.ExecuteWriter(context, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

type AssignmentMeal struct {
	Label  string
	Dishes []AssignedRecipeView
	Name   string
}

type AssignmentDay struct {
	Label string
	Date  time.Time
	Meals []AssignmentMeal
}

type AssignmentWeek struct {
	Start time.Time
	End   time.Time
	Days  []AssignmentDay
}

func ViewAssignedRecipes(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, res http.ResponseWriter) {
	query := req.URL.Query()

	user := getUser(db, session)
	if user == nil {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var start time.Time
	var err error

	fromDate := query.Get("from")
	if fromDate == "" {
		start = time.Now()

		daysOff := user.WeekStartDay - int64(start.Weekday())
		if daysOff > 0 {
			// This happens if today is Sunday and user prefers Monday.
			daysOff = daysOff - 7
		}

		start = start.Add(time.Duration(daysOff) * time.Hour * 24)
		fromDate = start.Format(dateFormat)
	} else {
		start, err = time.Parse(dateFormat, fromDate)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
	}

	toDate := query.Get("to")
	if toDate == "" {
		t := time.Now().AddDate(0, 0, 30)
		toDate = t.Format(dateFormat)
	}

	prevStart := start.AddDate(0, 0, -7)
	nextStart := start.AddDate(0, 0, 7)

	recipes := []AssignedRecipeView{}
	_, err = db.Select(&recipes,
		"SELECT a.id, a.recipe_id, a.date, a.meal, "+
			"r.fixed, r.name, r.source "+
			"FROM assignments a, recipes r "+
			"WHERE a.recipe_id=r.id AND a.date>=? AND a.date<? AND a.owner_id=?",
		fromDate, toDate, familyId)
	if err != nil {
		logError(err)
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}

	meals := map[string]*AssignmentMeal{}
	weeks := make([]AssignmentWeek, 4)
	for i := 0; i < 4; i++ {
		weeks[i] = AssignmentWeek{}
		weeks[i].Days = make([]AssignmentDay, 7)
		weeks[i].Start = start
		weeks[i].End = start.AddDate(0, 0, 6)
		for j := 0; j < 7; j++ {
			weeks[i].Days[j] = AssignmentDay{}
			weeks[i].Days[j].Date = start

			weeks[i].Days[j].Meals = make([]AssignmentMeal, 3)
			weeks[i].Days[j].Meals[0] = AssignmentMeal{
				Name:   "breakfast",
				Label:  "B",
				Dishes: make([]AssignedRecipeView, 0),
			}
			meals[start.Format(dateFormat)+"breakfast"] = &weeks[i].Days[j].Meals[0]

			weeks[i].Days[j].Meals[1] = AssignmentMeal{
				Name:   "lunch",
				Label:  "L",
				Dishes: make([]AssignedRecipeView, 0),
			}
			meals[start.Format(dateFormat)+"lunch"] = &weeks[i].Days[j].Meals[1]

			weeks[i].Days[j].Meals[2] = AssignmentMeal{
				Name:   "dinner",
				Label:  "D",
				Dishes: make([]AssignedRecipeView, 0),
			}
			meals[start.Format(dateFormat)+"dinner"] = &weeks[i].Days[j].Meals[2]

			start = start.AddDate(0, 0, 1)
		}
	}

	for _, recipe := range recipes {
		index := recipe.Date + recipe.Meal
		if entry, ok := meals[index]; ok {
			entry.Dishes = append(entry.Dishes, recipe)
		}
	}

	accountStatus, expired := getAccountStatus(db, session)

	context := pongo2.Context{
		"AccountStatus": accountStatus,
		"Expired":       expired,
		"Date":          time.Now().Format(dateFormat),
		"User":          user,
		"Families":      getFamilies(db, session),
		"FamilyId":      familyId,
		"Recipes":       recipes,
		"Weeks":         weeks,
		"NextStart":     nextStart.Format(dateFormat),
		"PrevStart":     prevStart.Format(dateFormat),
	}

	view, err := pongo2.FromCache("templates/view.html")
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = view.ExecuteWriter(context, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func ViewEditPage(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, res http.ResponseWriter) {
	query := req.URL.Query()

	user := getUser(db, session)
	if user == nil {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var start time.Time
	var end time.Time
	var err error

	fromDate := query.Get("from")
	if fromDate == "" {
		start = time.Now()
		fromDate = start.Format(dateFormat)
	} else {
		start, err = time.Parse(dateFormat, fromDate)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
	}

	toDate := query.Get("to")
	if toDate == "" {
		end = time.Now().AddDate(0, 0, 30)
		toDate = end.Format(dateFormat)
	} else {
		end, err = time.Parse(dateFormat, toDate)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
	}

	prevStart := start.AddDate(0, 0, -7)
	nextEnd := end.AddDate(0, 0, 7)

	recipes := []AssignedRecipeView{}
	_, err = db.Select(&recipes,
		"SELECT a.id, a.recipe_id, a.date, a.meal, "+
			"r.fixed, r.name, r.source "+
			"FROM assignments a, recipes r "+
			"WHERE a.recipe_id=r.id AND a.date>=? AND a.date<? AND a.owner_id=?",
		fromDate, toDate, familyId)
	if err != nil {
		logError(err)
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}

	meals := map[string]*AssignmentMeal{}
	days := make([]*AssignmentDay, 0)
	for start.Before(end) {
		day := &AssignmentDay{
			Date:  start,
			Meals: make([]AssignmentMeal, 3),
		}

		day.Meals[0] = AssignmentMeal{
			Name:   "breakfast",
			Label:  "B",
			Dishes: make([]AssignedRecipeView, 0),
		}
		meals[start.Format(dateFormat)+"breakfast"] = &day.Meals[0]

		day.Meals[1] = AssignmentMeal{
			Name:   "lunch",
			Label:  "L",
			Dishes: make([]AssignedRecipeView, 0),
		}
		meals[start.Format(dateFormat)+"lunch"] = &day.Meals[1]

		day.Meals[2] = AssignmentMeal{
			Name:   "dinner",
			Label:  "D",
			Dishes: make([]AssignedRecipeView, 0),
		}
		meals[start.Format(dateFormat)+"dinner"] = &day.Meals[2]

		days = append(days, day)
		start = start.AddDate(0, 0, 1)
	}

	for _, recipe := range recipes {
		index := recipe.Date + recipe.Meal
		if entry, ok := meals[index]; ok {
			entry.Dishes = append(entry.Dishes, recipe)
		}
	}

	available := []Recipe{}
	_, err = db.Select(&available,
		"SELECT *"+
			"FROM recipes "+
			"WHERE show=1 AND owner_id=?",
		familyId)
	if err != nil {
		logError(err)
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}

	accountStatus, expired := getAccountStatus(db, session)

	context := pongo2.Context{
		"AccountStatus": accountStatus,
		"Expired":       expired,
		"Date":          time.Now().Format(dateFormat),
		"User":          user,
		"FamilyId":      familyId,
		"Families":      getFamilies(db, session),
		"Available":     available,
		"Days":          days,
		"From":          fromDate,
		"To":            toDate,
		"PrevStart":     prevStart.Format(dateFormat),
		"NextEnd":       nextEnd.Format(dateFormat),
	}

	view, err := pongo2.FromCache("templates/edit.html")
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = view.ExecuteWriter(context, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

type AssignmentGroup struct {
	Date        time.Time
	Ingredients []AssignedIngredientView
}

type AssignmentSection struct {
	Id            int64
	Name          string
	Groups        map[string]*AssignmentGroup
	OrderedGroups []*AssignmentGroup
}

func ViewShoppingPage(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, res http.ResponseWriter) {
	query := req.URL.Query()
	var err error

	user := getUser(db, session)
	if user == nil {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var start time.Time
	var end time.Time

	fromDate := query.Get("from")
	if fromDate == "" {
		start = time.Now()
		fromDate = start.Format(dateFormat)
	} else {
		start, err = time.Parse(dateFormat, fromDate)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
	}

	toDate := query.Get("to")
	if toDate == "" {
		end = time.Now().AddDate(0, 0, 30)
		toDate = end.Format(dateFormat)
	} else {
		end, err = time.Parse(dateFormat, toDate)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
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
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}

	for i, item := range assignments {
		if !item.ClassId.Valid {
			class_id := Classifier.Classify(item.Name, item.Amount.String)
			item.setClassLocal(db, class_id)
			assignments[i] = item
		}
	}

	sections := make(map[string]*AssignmentSection)
	for _, item := range assignments {
		class := item.Class.String
		if class == "" {
			class = "Miscellaneous"
		}

		section, ok := sections[class]
		if !ok {
			sections[class] = &AssignmentSection{
				Id:            item.ClassId.Int64,
				Name:          class,
				Groups:        make(map[string]*AssignmentGroup),
				OrderedGroups: make([]*AssignmentGroup, 0),
			}
			section = sections[class]
		}

		group, ok := section.Groups[item.Date]
		if !ok {
			date, _ := time.Parse(dateFormat, item.Date)
			section.Groups[item.Date] = &AssignmentGroup{
				Date:        date,
				Ingredients: make([]AssignedIngredientView, 0),
			}
			group = section.Groups[item.Date]
		}

		group.Ingredients = append(group.Ingredients, item)
	}

	for _, section := range sections {
		for _, group := range section.Groups {
			section.OrderedGroups = append(section.OrderedGroups, group)
		}
		sort.Slice(section.OrderedGroups, func(i, j int) bool {
			return section.OrderedGroups[i].Date.Before(section.OrderedGroups[j].Date)
		})
	}

	accountStatus, expired := getAccountStatus(db, session)

	context := pongo2.Context{
		"AccountStatus": accountStatus,
		"Expired":       expired,
		"Date":          time.Now().Format(dateFormat),
		"User":          user,
		"FamilyId":      familyId,
		"Families":      getFamilies(db, session),
		"Sections":      sections,
		"From":          fromDate,
		"To":            toDate,
	}

	view, err := pongo2.FromCache("templates/shopping.html")
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = view.ExecuteWriter(context, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func ViewRecipesPage(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, res http.ResponseWriter) {
	user := getUser(db, session)
	if user == nil {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := req.URL.Query()
	importId := query.Get("import_id")

	available := []Recipe{}
	_, err := db.Select(&available,
		"SELECT *"+
			"FROM recipes "+
			"WHERE show=1 AND owner_id=?"+
			"AND (? OR import_id=?)",
		familyId, importId == "", importId)
	if err != nil {
		logError(err)
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}

	accountStatus, expired := getAccountStatus(db, session)

	context := pongo2.Context{
		"AccountStatus": accountStatus,
		"Expired":       expired,
		"Date":          time.Now().Format(dateFormat),
		"User":          user,
		"FamilyId":      familyId,
		"Families":      getFamilies(db, session),
		"Recipes":       available,
	}

	view, err := pongo2.FromCache("templates/recipe.html")
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = view.ExecuteWriter(context, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func ImportAndViewRecipesPage(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, res http.ResponseWriter) {
	user := getUser(db, session)
	if user == nil {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	file, _, err := req.FormFile("file")
	defer file.Close()

	if err != nil {
		logError(err)
		http.Error(res, "Bad Request", http.StatusBadRequest)
		return
	}

	query := req.URL.Query()

	// ImportId is passed by the client for the user's convenience.
	// We will set all recipes in this batch to the given ImportId.
	importId, _ := strconv.ParseInt(query.Get("import_id"), 10, 64)

	addedRecipes := []Recipe{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		recipe := Recipe{
			OwnerId:  familyId,
			ImportId: importId,
			Show:     true,
			Fixed:    false,
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
			http.Error(res, "Server Error", http.StatusInternalServerError)
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
					http.Error(res, "Server Error", http.StatusInternalServerError)
					return
				}
			}
		}

		addedRecipes = append(addedRecipes, recipe)
	}

	accountStatus, expired := getAccountStatus(db, session)

	context := pongo2.Context{
		"AccountStatus": accountStatus,
		"Expired":       expired,
		"Date":          time.Now().Format(dateFormat),
		"User":          user,
		"FamilyId":      familyId,
		"Families":      getFamilies(db, session),
		"Recipes":       addedRecipes,
	}

	view, err := pongo2.FromCache("templates/recipe.html")
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = view.ExecuteWriter(context, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func ViewRecipePage(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, res http.ResponseWriter) {
	user := getUser(db, session)
	if user == nil {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	familyId, ok := checkFamilyParam(params, session)
	if !ok {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	recipe_id, err := strconv.Atoi(params["recipe_id"])
	if err != nil {
		logError(err)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := db.Get(Recipe{}, recipe_id)
	if err != nil {
		logError(err)
		http.Error(res, "Not Found", http.StatusNotFound)
		return
	} else if result == nil {
		http.Error(res, "Not Found", http.StatusNotFound)
		return
	}

	recipe, ok := result.(*Recipe)
	if !ok {
		http.Error(res, "Error", http.StatusInternalServerError)
		return
	}

	if recipe.OwnerId != familyId {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ingredients := []Ingredient{}
	_, err = db.Select(&ingredients,
		"SELECT * FROM ingredients WHERE owner_id=? AND recipe_id=?",
		familyId, recipe_id)
	if err != nil {
		logError(err)
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}

	available := []Recipe{}
	_, err = db.Select(&available,
		"SELECT *"+
			"FROM recipes "+
			"WHERE show=1 AND owner_id=?",
		familyId)
	if err != nil {
		logError(err)
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}

	accountStatus, expired := getAccountStatus(db, session)

	context := pongo2.Context{
		"AccountStatus": accountStatus,
		"Expired":       expired,
		"Date":          time.Now().Format(dateFormat),
		"User":          user,
		"FamilyId":      familyId,
		"Families":      getFamilies(db, session),
		"Recipes":       available,
		"Recipe":        recipe,
		"Ingredients":   ingredients,
	}

	view, err := pongo2.FromCache("templates/recipe.html")
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = view.ExecuteWriter(context, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func ViewProfilePage(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, res http.ResponseWriter) {
	user := getUser(db, session)
	if user == nil {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accountStatus, expired := getAccountStatus(db, session)

	context := pongo2.Context{
		"AccountStatus": accountStatus,
		"Expired":       expired,
		"Date":          time.Now().Format(dateFormat),
		"User":          user,
		"Families":      getFamilies(db, session),
		"ImportId":      time.Now().Unix(),
	}

	view, err := pongo2.FromCache("templates/profile.html")
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = view.ExecuteWriter(context, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func ViewVerifiedPage(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, res http.ResponseWriter) {
	user := getUser(db, session)
	if user == nil {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := req.URL.Query()
	token := query.Get("token")

	if token == user.EmailToken {
		user.EmailVerified.SetValid(true)

		_, err := db.Update(user)
		if err != nil {
			logError(err)
			http.Error(res, "Server Error", http.StatusInternalServerError)
			return
		}
	}

	ViewProfilePage(db, params, session, req, res)
}

func AdminViewUsers(db *gorp.DbMap, params martini.Params, session sessions.Session, req *http.Request, res http.ResponseWriter) {
	var err error

	user := getUser(db, session)
	if user == nil || !user.Admin.Bool {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	families := []Family{}
	_, err = db.Select(&families, "SELECT * FROM families")
	if err != nil {
		logError(err)
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}

	users := []User{}
	_, err = db.Select(&users, "SELECT * FROM users")
	if err != nil {
		logError(err)
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}

	context := pongo2.Context{
		"Date":     time.Now().Format(dateFormat),
		"Families": families,
		"User":     user,
		"Users":    users,
	}

	view, err := pongo2.FromCache("templates/admin/users.html")
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = view.ExecuteWriter(context, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
