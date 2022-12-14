package main

import (
	"log"
	"net/http"

	"github.com/duo-labs/webauthn.io/session"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/gorilla/mux"
)

var webAuthn *webauthn.WebAuthn
var sessionStore *session.Store
var userDB *userdb

func main() {
	
	r := mux.NewRouter()

	r.HandleFunc("/register/begin/{username}", BeginRegistration).Methods("GET")
	r.HandleFunc("/register/finish/{username}", FinishRegistration).Methods("POST")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./")))

	var err error
	webAuthn, err = webauthn.New(&webauthn.Config{
		RPDisplayName: "Foobar Corp.",     // display name for your site
		RPID:          "localhost",        // generally the domain name for your site
	})

	if err != nil {
		log.Fatal("failed to create WebAuthn from config:", err)
	}

	userDB = DB()

	sessionStore, err = session.NewStore()
	if err != nil {
		log.Fatal("failed to create session store:", err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/register/begin/{username}", BeginRegistration).Methods("GET")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./")))

	serverAddress := ":8080"
	log.Println("starting server at", serverAddress)
	log.Fatal(http.ListenAndServe(serverAddress, r))

	func BeginRegistration(w http.ResponseWriter, r *http.Request) {

		// get username
		vars := mux.Vars(r)
		username, ok := vars["username"]
		if !ok {
			jsonResponse(w, fmt.Errorf("must supply a valid username i.e. foo@bar.com"), http.StatusBadRequest)
			return
		}
	
		// get user
		user, err := userDB.GetUser(username)
		// user doesn't exist, create new user
		if err != nil {
			displayName := strings.Split(username, "@")[0]
			user = NewUser(username, displayName)
			userDB.PutUser(user)
		}
	
		// generate PublicKeyCredentialCreationOptions, session data
		options, sessionData, err := webAuthn.BeginRegistration(
			user,
		)
	
		if err != nil {
			log.Println(err)
			jsonResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
	
		// store session data as marshaled JSON
		err = sessionStore.SaveWebauthnSession("registration", sessionData, r, w)
		if err != nil {
			log.Println(err)
			jsonResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
	
		jsonResponse(w, options, http.StatusOK)
	}
	

	func jsonResponse(w http.ResponseWriter, d interface{}, c int) {
		dj, err := json.Marshal(d)
		if err != nil {
			http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(c)
		fmt.Fprintf(w, "%s", dj)
	}

	func FinishRegistration(w http.ResponseWriter, r *http.Request) {

		// get username
		vars := mux.Vars(r)
		username := vars["username"]
	
		// get user
		user, err := userDB.GetUser(username)
		// user doesn't exist
		if err != nil {
			log.Println(err)
			jsonResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
	
		// load the session data
		sessionData, err := sessionStore.GetWebauthnSession("registration", r)
		if err != nil {
			log.Println(err)
			jsonResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
	
		credential, err := webAuthn.FinishRegistration(user, sessionData, r)
		if err != nil {
			log.Println(err)
			jsonResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
	
		user.AddCredential(*credential)
	
		jsonResponse(w, "Registration Success", http.StatusOK)
		return
	}
	
	
	
}
