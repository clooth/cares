package main

import (
	"database/sql"
	"encoding/csv"
	"html/template"
	"os"
	"time"
)

type Import struct {
	Id uint64
	PostId uint64
	Source string
	Identifier string
}

func NewImport() (*Import) {
	return &Import{0, 0, "", ""}
}

func ImportBySourceIdentifier(source, identifier string) (*Import, error) {
	row := db.QueryRow("SELECT id, post, source, identifier FROM import WHERE source = $1 AND identifier = $2 LIMIT 1",
		source, identifier)

	var id, postid uint64
	err := row.Scan(&id, &postid, &source, &identifier)
	if err != nil {
		return nil, err
	}

	i := &Import{id, postid, source, identifier}
	return i, nil
}

func (i *Import) Save() (err error) {
	if i.Id == 0 {
		row := db.QueryRow("INSERT INTO import (post, source, identifier) values ($1, $2, $3) RETURNING id",
			i.PostId, i.Source, i.Identifier)
		var id uint64
		err = row.Scan(&id)
		if err != nil {
			return
		}
		i.Id = id
	} else {
		_, err = db.Exec("UPDATE import SET post = $2, source = $3, identifier = $4 WHERE id = $1",
			i.Id, i.PostId, i.Source, i.Identifier)
	}
	return
}

func ImportThinkup(path string) {
	logr.Debugln("Importing from Thinkup export", path)
	port, err := os.Open(path)
	if err != nil {
		logr.Errln("Error opening", path, "for import:", err.Error())
		return
	}

	r := csv.NewReader(port)
	// There may be missing header columns, so turn off field count checking.
	r.FieldsPerRecord = -1

	head, err := r.Read()
	if err != nil {
		logr.Errln("Error reading from import file", path, ":", err.Error())
		return
	}

	count := 0
	for {
		record, err := r.Read()
		if err != nil {
			break
		}

		data := make(map[string]string)
		for i, field := range head {
			data[field] = record[i]
		}

		// TODO: import replies, once there's something reasonable to import them as.
		if data["in_reply_to_post_id"] != "" {
			logr.Debugln("Skipping post (twitter,", data["post_id"], ") as it is a reply")
			continue
		}
		// TODO: import repeats, once there's something reasonable to import them as.
		if data["in_retweet_of_post_id"] != "" {
			logr.Debugln("Skipping post (twitter,", data["post_id"], ") as it is a repeat")
			continue
		}

		// okay now what
		im, err := ImportBySourceIdentifier("twitter", data["post_id"])
		if err == sql.ErrNoRows {
			im = NewImport()
			im.Source = "twitter"
			im.Identifier = data["post_id"]
		} else if err != nil {
			logr.Errln("Error searching for existing imported post (twitter,", data["post_id"], "):", err.Error())
			return
		}

		var post *Post
		if im.PostId != 0 {
			post, err = PostById(im.PostId)
			if err != nil {
				logr.Errln("Error loading already-imported post", im.PostId, "for twitter post", im.Identifier, ":", err.Error())
				return
			}
		} else {
			post = NewPost()
		}

		post.Posted, err = time.Parse("2006-01-02 15:04:05", data["pub_date"])
		if err != nil {
			logr.Errln("Error parsing publish time", data["pub_date"], "for twitter post", data["post_id"], ":", err.Error())
			return
		}

		// TODO: make the links link.
		post.Html = template.HTMLEscapeString(data["post_text"])

		// TODO: store the source?
		// TODO: store the geoplace

		err = post.Save()
		if err != nil {
			logr.Errln("Error saving imported post:", err.Error())
			return
		}

		im.PostId = post.Id
		err = im.Save()
		if err != nil {
			logr.Errln("Error saving import notation for post", im.PostId, ":", err.Error())
			return
		}

		logr.Debugln("Imported post (twitter,", im.Identifier, ")")
		count++
	}
	if err != nil {
		logr.Errln("Error reading import records:", err.Error())
		return
	}

	logr.Debugln("Finished importing", count, "posts!")
}
