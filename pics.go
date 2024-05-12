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
	"strconv"
	"math"
)

type Image struct {
	Id int
	Size int
	Type string
	Path string
	Year int
	
}

type Question struct {
	Path string
	QuestionNum int
}

type upfile struct {
	ID    int
	Fname string
	Fsize string
	Ftype string
	Path  string
	Count int
}

type answer struct {
	Path string
	QuestionNum int
	Year int
	Score int
	Next int
}


var currentImages [5]int = [5]int{2,3,4,0,0}
var validPath = regexp.MustCompile("^/(edit|save|view|static|question|questionAns)/([a-zA-Z0-9]+)$")
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

func newGame(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	db := dbConn()
	selDB, err := db.Query("SELECT t1.id, fsize, ftype, year, path FROM upload AS t1 JOIN (SELECT id FROM upload ORDER BY RAND() LIMIT 5) as t2 ON t1.id=t2.id")

	if err != nil {
		panic(err.Error())
	}

	upld := upfile{}
	res := []upfile{}
	i := 0
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
		currentImages[i] = upld.ID
		i+=1

	}
	http.Redirect(w, r, "/question/1", 301)
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

		http.Redirect(w, r, "/upload", 301)
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
	http.Redirect(w, r, "/upload", 301)
}

func imageHandler(w http.ResponseWriter, r *http.Request, img Image, q int) {
	question := &Question{Path:img.Path, QuestionNum: q}
	err := templates.ExecuteTemplate(w, "image.html", question)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func ImageById(id int) (Image, error){

	db := dbConn()

	var img Image
	row := db.QueryRow("SELECT * FROM upload WHERE Id =?", id)
	log.Printf("%v", row)
	
	if err := row.Scan(&img.Id, &img.Size, &img.Type, &img.Year, &img.Path); err != nil{
		if err == sql.ErrNoRows{
			return img, fmt.Errorf("no such image %d", id)
		}
		return img, fmt.Errorf("image %d: %v", id, err)
	}

	return img, nil
}

func answerHandler(w http.ResponseWriter, r *http.Request, year int, q int) {
	db := dbConn()
	id := currentImages[q-1]
	var img Image
	row := db.QueryRow("SELECT * FROM upload WHERE Id =?", id)
	log.Printf("%v", row)
	
	if err := row.Scan(&img.Id, &img.Size, &img.Type, &img.Year, &img.Path); err != nil{
		if err == sql.ErrNoRows{
			http.NotFound(w, r)
			log.Printf("no such image %d", id)
			return
			
		}
		http.NotFound(w, r)
		log.Printf("image %d: %v", id, err)
		return
		
	}

	score := 25 - math.Abs(float64(year-img.Year))

	ans := &answer{img.Path, q, img.Year, int(score), q+1}

	err := templates.ExecuteTemplate(w, "answer.html", ans)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func makeAnswer(fn func(http.ResponseWriter, *http.Request, int, int)) http.HandlerFunc{
	return func(w http.ResponseWriter, r *http.Request){
		
		ind, err := strconv.Atoi(validPath.FindStringSubmatch(r.URL.Path)[2])
		
		if err != nil{
			http.NotFound(w, r)
			return
		}
		
		year, err := strconv.Atoi(r.FormValue("year"))
		if err != nil{
			http.NotFound(w, r)
			return
		}

		fn(w, r, year,ind)
	}
}

func makeImage(fn func(http.ResponseWriter, *http.Request, Image, int)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		ind, err := strconv.Atoi(validPath.FindStringSubmatch(r.URL.Path)[2])
		if ind > 5 {
			http.Redirect(w, r, "/newGame", 301)
			return
		}
		
		if err != nil{
			http.NotFound(w, r)
			return
		}
		id := currentImages[ind-1]
		

		img, err := ImageById(id)
		log.Print( err)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		/*
			if m == nil {
				http.NotFound(w, r)
				return
			}*/
		fn(w, r, img, ind)
	}
}

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static", fs))

	http.HandleFunc("/dele", delete)
	http.HandleFunc("/image/", makeImage(imageHandler))
	http.HandleFunc("/uploadfiles", uploadFiles)
	http.HandleFunc("/question/", makeImage((imageHandler)))
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/questionAns/", makeAnswer(answerHandler))
	http.HandleFunc("/newGame", newGame)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
