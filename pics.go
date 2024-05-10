package main

import (
	"html/template"
	"log"
	"io"
	"fmt"
	"net/http"
	"regexp"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"os"
)

type Image struct {
	QNum int
	Path string
}

type upfile struct {
	ID    int
	Fname string
	Fsize string
	Ftype string
	Path  string
	Count int
}


var currentImages [5]int = [5]int{0,0,0,0,0}
var validPath = regexp.MustCompile("^/(edit|save|view|static)/([a-zA-Z0-9]+)$")
var templates = template.Must(template.ParseGlob("templates/*"))

func dbConn() (db *sql.DB) {

	er := godotenv.Load(".env")
	if er != nil {
		panic(er.Error())
	}
	dbDriver := os.Getenv("DB_Driver")
	dbUser := os.Getenv("DB_User")
	dbPass := os.Getenv("DB_Password")
	dbName := os.Getenv("DB_Name")
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	return db
}

func upload(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	selDB, err := db.Query("SELECT * FROM upload ORDER BY id DESC")
	if err != nil {
		panic(err.Error())
	}
	upld := upfile{}
	res := []upfile{}
	for selDB.Next() {
		var id int
		var fname, fsize, ftype, path string
		err = selDB.Scan(&id, &fname, &fsize, &ftype, &path)
		if err != nil {
			panic(err.Error())
		}
		upld.ID = id
		upld.Fname = fname
		upld.Fsize = fsize
		upld.Ftype = ftype
		upld.Path = path
		res = append(res, upld)

	}

	upld.Count = len(res)

	if upld.Count > 0 {
		templates.ExecuteTemplate(w, "uploadfile.html", res)
	} else {
		templates.ExecuteTemplate(w, "uploadfile.html", nil)
	}

	db.Close()

}

func uploadFiles(w http.ResponseWriter, r *http.Request) {
	// tmpl.ExecuteTemplate(w, "uploadfile.html", r)
	db := dbConn()
	// Maximum upload of 10 MB files
	r.ParseMultipartForm(200000)
	if r == nil {
		fmt.Fprintf(w, "No files can be selected\n")
	}
	// ok, no problem so far, read the Form data
	formdata := r.MultipartForm

	//get the *fileheaders
	fil := formdata.File["files"] // grab the filenames
	year := r.FormValue("year")

	for i := range fil { // loop through the files one by one

		//file save to open
		file, err := fil[i].Open()
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		defer file.Close()

		//fname := fil[i].Filename
		fsize := fil[i].Size
		kilobytes := fsize / 1024
		// megabytes := (float64)(kilobytes / 1024) // cast to type float64

		ftype := fil[i].Header.Get("Content-type")

		// Create file

		tempFile, err := os.CreateTemp("static/Images/uploadimage/", "upload-*.jpg")
		if err != nil {
			fmt.Println(err)
		}
		defer tempFile.Close()
		filepath := tempFile.Name()

		// read all of the contents of our uploaded file into a
		// byte array
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			fmt.Println(err)
		}
		// write this byte array to our temporary file
		tempFile.Write(fileBytes)

		// return that we have successfully uploaded our file!

		insForm, err := db.Prepare("INSERT INTO upload(fsize, ftype, path, year) VALUES(?,?,?,?)")
		if err != nil {
			panic(err.Error())
		} else {
			log.Println("data insert successfully . . .")
		}
		insForm.Exec(kilobytes, ftype, filepath, year)

		log.Printf("Successfully Uploaded File\n")
		defer db.Close()

		http.Redirect(w, r, "/", 301)
	}
}

func delete(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	emp := r.URL.Query().Get("id")
	delForm, err := db.Prepare("DELETE FROM upload WHERE id=?")
	if err != nil {
		panic(err.Error())
	}
	delForm.Exec(emp)
	log.Println("deleted successfully")
	defer db.Close()
	http.Redirect(w, r, "/", 301)
}

func imageHandler(w http.ResponseWriter, r *http.Request, path string, qNum int) {
	q := &Image{QNum: qNum, Path: path}
	err := templates.ExecuteTemplate(w, "image.html", q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func makeImage(fn func(http.ResponseWriter, *http.Request, string, int)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db := dbConn()
		selDB, err := db.Query("SELECT * FROM upload ORDER BY id DESC")
		path := "http://192.168.2.245:8080/static/branch01.png"
		/*
			if m == nil {
				http.NotFound(w, r)
				return
			}*/
		fn(w, r, path, 1)
	}
}

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static", fs))

	http.HandleFunc("/dele", delete)
	http.HandleFunc("/image/", makeImage(imageHandler))
	http.HandleFunc("/uploadfiles", uploadFiles)
	http.HandleFunc("/question", makeImage((imageHandler)))
	http.HandleFunc("/upload", upload)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
