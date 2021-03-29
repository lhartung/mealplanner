package backend

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/gorp.v2"
)

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

var Classifier *classifier

func configureDB(db *sql.DB) {
	// synchronous=OFF increases the risk for data loss if the server crashes,
	// but it was the easiest way to get the write performance up to an
	// acceptable level.
	commands := `
		PRAGMA synchronous = NORMAL;
	`

	_, err := db.Exec(commands)
	panicOnErr(err)
}

func expiresHeader() string {
	return time.Now().Add(time.Hour * 24).Format(http.TimeFormat)
}

func Run() {
	db, err := sql.Open("sqlite3", "mealplanner.db")
	panicOnErr(err)
	defer db.Close()

	configureDB(db)

	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	dbmap.AddTableWithName(Migration{}, "migrations").SetKeys(true, "Id")
	dbmap.AddTableWithName(Family{}, "families").SetKeys(true, "Id")
	dbmap.AddTableWithName(FamilyMember{}, "familymembers").SetKeys(true, "Id")
	dbmap.AddTableWithName(User{}, "users").SetKeys(true, "Id")
	dbmap.AddTableWithName(Recipe{}, "recipes").SetKeys(true, "Id")
	dbmap.AddTableWithName(Ingredient{}, "ingredients").SetKeys(true, "Id")
	dbmap.AddTableWithName(IngredientClass{}, "ingclasses").SetKeys(true, "Id")
	dbmap.AddTableWithName(Assignment{}, "assignments").SetKeys(true, "Id")
	err = dbmap.CreateTablesIfNotExists()
	panicOnErr(err)

    MigrateDatabase(dbmap)

	Classifier = NewClassifier()
	Classifier.Train(dbmap)

	mainRouter := martini.NewRouter()
	m := martini.New()

	m.Use(martini.Logger())
	m.Use(martini.Recovery())
	m.MapTo(mainRouter, (*martini.Routes)(nil))
	m.Action(mainRouter.Handle)

	m.Use(martini.Static("public", martini.StaticOptions{
		Expires: expiresHeader,
	}))

	// Setup middleware
	// We are only using this Renderer for JSON. It will automatically load
	// .tmpl files from the templates directory, but we do not want that.
	m.Use(render.Renderer(render.Options{
		Extensions: []string{".tmpl"},
	}))

	// Set Cache-Control header for all responses, since most are private.
	m.Use(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Cache-Control", "private, no-store, max-age=0")
	})

	store := sessions.NewCookieStore([]byte("wieth5cie2sheuRieTi5lahh3aih4aerugh5faelaim8shi2nophis2iekuth7Ae"))
	m.Use(sessions.Sessions("session", store))

	checkUser := func(session sessions.Session, w http.ResponseWriter, r *http.Request) {
		userId := session.Get("UserId")
		if userId == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}

	checkAdmin := func(session sessions.Session, w http.ResponseWriter, r *http.Request) {
		admin := session.Get("Admin")
		if admin != true {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}

	// Dynamic pages
	mainRouter.Get("/", ViewLandingPage)
	mainRouter.Get("/index.html", ViewLandingPage)
	mainRouter.Get("/landing.html", ViewLandingPage)
	mainRouter.Get("/sign-up.html", ViewSignUpPage)
	mainRouter.Get("/sign-out.html", ViewSignOutPage)
	mainRouter.Get("/family/:family_id/view.html", ViewAssignedRecipes)
	mainRouter.Get("/family/:family_id/edit.html", ViewEditPage)
	mainRouter.Get("/family/:family_id/shopping.html", ViewShoppingPage)
	mainRouter.Get("/family/:family_id/recipes.html", ViewRecipesPage)
	mainRouter.Post("/family/:family_id/recipes.html", ImportAndViewRecipesPage)
	mainRouter.Get("/family/:family_id/recipe/:recipe_id/edit.html", ViewRecipePage)
	mainRouter.Get("/user/:user_id/profile.html", ViewProfilePage)
	mainRouter.Get("/user/:user_id/verify.html", ViewVerifiedPage)

	// Admin pages
	mainRouter.Group("/admin", func(r martini.Router) {
		r.Get("/users.html", AdminViewUsers)
	}, checkAdmin)

	// Unauthenticated routes
	mainRouter.Group("/api", func(r martini.Router) {
		r.Post("/login", GetToken)
		r.Post("/logout", ClearToken)
		r.Post("/register", CreateAccount)
	})

	// Authenticated routes
	mainRouter.Group("/api", func(r martini.Router) {
		r.Get("/families", ListFamilies)
		r.Get("/families/:family_id", GetFamily)
		r.Put("/families/:family_id", UpdateFamily)

		r.Get("/users/:id", GetUser)
		r.Put("/users/:id", UpdateUser)
		r.Put("/users/:id/password", UpdateUserPassword)
		r.Post("/users/:id/verification", SendEmailVerification)

		r.Group("/family/:family_id", func(family martini.Router) {
			family.Get("/members", ListMembers)
			family.Post("/members", CreateMember)
			family.Delete("/members/:id", DeleteMember)

			family.Get("/recipes", ListRecipes)
			family.Post("/recipes", CreateRecipe)
			family.Get("/recipes/assigned", ListAssignedRecipes)
			family.Get("/recipes/:id", GetRecipe)
			family.Put("/recipes/:id", UpdateRecipe)
			family.Delete("/recipes/:id", DeleteRecipe)
			family.Post("/recipes/import", ImportRecipes)

			family.Get("/ingredients", ListIngredients)
			family.Post("/ingredients", CreateIngredient)
			family.Get("/ingredients/assigned", ListAssignedIngredients)
			family.Get("/ingredients/:id", GetIngredient)
			family.Put("/ingredients/:id", UpdateIngredient)
			family.Delete("/ingredients/:id", DeleteIngredient)

			family.Get("/assignments", ListAssignments)
			family.Post("/assignments", CreateAssignment)
			family.Get("/assignments/:id", GetAssignment)
			family.Put("/assignments/:id", UpdateAssignment)
			family.Delete("/assignments/:id", DeleteAssignment)
		})
	}, checkUser)

	// Make db available to handlers.
	m.Map(dbmap)

	m.Run()
}
