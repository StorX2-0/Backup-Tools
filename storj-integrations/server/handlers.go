package server

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	google "storj-integrations/apps/google"
	"storj-integrations/storage"
	"storj-integrations/storj"
	"storj-integrations/utils"

	"github.com/labstack/echo/v4"
)

// <<<<<------------ GOOGLE DRIVE ------------>>>>>

// Sends file from Google Drive to Storj
func handleSendFileFromGoogleDriveToStorj(c echo.Context) error {
	id := c.Param("ID")

	name, data, err := google.GetFile(c, id)
	if err != nil {
		return c.String(http.StatusForbidden, "error")
	}
	accesGrant, err := c.Cookie("storj_access_token")
	if err != nil {
		return c.String(http.StatusForbidden, "storj is unauthenticated")
	}

	err = storj.UploadObject(context.Background(), accesGrant.Value, "google-drive", name, data)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	return c.String(http.StatusOK, "file "+name+" was successfully uploaded from Google Drive to Storj")
}

// Sends file from Storj to Google Drive
func handleSendFileFromStorjToGoogleDrive(c echo.Context) error {
	name := c.Param("name")
	accesGrant, err := c.Cookie("storj_access_token")
	if err != nil {
		return c.String(http.StatusForbidden, "storj is unauthenticated")
	}

	data, err := storj.DownloadObject(context.Background(), accesGrant.Value, "google-drive", name)
	if err != nil {
		return c.String(http.StatusForbidden, "error downloading object from Storj"+err.Error())
	}

	err = google.UploadFile(c, name, data)
	if err != nil {
		return c.String(http.StatusForbidden, "error uploading file to Google Drive")
	}

	return c.String(http.StatusOK, "file "+name+" was successfully uploaded from Storj to Google Drive")
}

// <<<<<------------ GOOGLE PHOTOS ------------>>>>>

type AlbumsJSON struct {
	Title string `json:"album_title"`
	ID    string `json:"album_id"`
	Items string `json:"items_count"`
}

// Shows list of user's Google Photos albums.
func handleListGPhotosAlbums(c echo.Context) error {
	client, err := google.NewGPhotosClient(c)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	albs, err := client.ListAlbums(c)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	var photosListJSON []*AlbumsJSON
	for _, v := range albs {
		photosListJSON = append(photosListJSON, &AlbumsJSON{
			Title: v.Title,
			ID:    v.ID,
			Items: v.MediaItemsCount,
		})
	}

	return c.JSON(http.StatusOK, photosListJSON)

}

type PhotosJSON struct {
	Name string `json:"file_name"`
	ID   string `json:"file_id"`
}

// Shows list of user's Google Photos items in given album.
func handleListPhotosInAlbum(c echo.Context) error {
	id := c.Param("ID")

	client, err := google.NewGPhotosClient(c)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	files, err := client.ListFilesFromAlbum(c, id)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	var photosRespJSON []*PhotosJSON
	for _, v := range files {
		photosRespJSON = append(photosRespJSON, &PhotosJSON{
			Name: v.Filename,
			ID:   v.ID,
		})
	}

	return c.JSON(http.StatusOK, photosRespJSON)
}

// Sends photo item from Storj to Google Photos.
func handleSendFileFromStorjToGooglePhotos(c echo.Context) error {
	name := c.Param("name")
	accesGrant, err := c.Cookie("storj_access_token")
	if err != nil {
		return c.String(http.StatusForbidden, "storj is unauthenticated")
	}

	data, err := storj.DownloadObject(context.Background(), accesGrant.Value, "google-photos", name)
	if err != nil {
		return c.String(http.StatusForbidden, "error downloading object from Storj"+err.Error())
	}

	path := filepath.Join("./cache", utils.CreateUserTempCacheFolder(), name)
	file, err := os.Create(path)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	file.Write(data)
	file.Close()
	defer os.Remove(path)

	client, err := google.NewGPhotosClient(c)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	err = client.UploadFileToGPhotos(c, name, "Storj Album")
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	return c.String(http.StatusOK, "file "+name+" was successfully uploaded from Storj to Google Photos")
}

// Sends photo item from Google Photos to Storj.
func handleSendFileFromGooglePhotosToStorj(c echo.Context) error {
	id := c.Param("ID")
	accesGrant, err := c.Cookie("storj_access_token")
	if err != nil {
		return c.String(http.StatusForbidden, "storj is unauthenticated")
	}

	client, err := google.NewGPhotosClient(c)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	item, err := client.GetPhoto(c, id)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	resp, err := http.Get(item.BaseURL)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	err = storj.UploadObject(context.Background(), accesGrant.Value, "google-photos", item.Filename, body)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	return c.String(http.StatusOK, "file "+item.Filename+" was successfully uploaded from Google Photos to Storj")
}

// <<<<<------------ GMAIL ------------>>>>>

type ThreadJSON struct {
	ID      string `json:"thread_id"`
	Snippet string `json:"snippet"`
}

// Fetches user threads, returns their IDs and snippets.
func handleGmailGetThreads(c echo.Context) error {
	GmailClient, err := google.NewGmailClient(c)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	threads, err := GmailClient.GetUserThreads("")
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	// TODO: implement next page token (now only first page is avialable)

	var jsonResp []*ThreadJSON
	for _, v := range threads.Threads {
		jsonResp = append(jsonResp, &ThreadJSON{
			ID:      v.ID,
			Snippet: v.Snippet,
		})
	}
	return c.JSON(http.StatusOK, jsonResp)
}

type MessageListJSON struct {
	ID       string `json:"message_id"`
	ThreadID string `json:"thread_id"`
}

// Fetches user messages, returns their ID's and threat's IDs.
func handleGmailGetMessages(c echo.Context) error {
	GmailClient, err := google.NewGmailClient(c)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	msgs, err := GmailClient.GetUserMessages("")
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	// TODO: implement next page token (now only first page is avialable)

	var jsonMessages []*MessageListJSON
	for _, v := range msgs.Messages {
		jsonMessages = append(jsonMessages, &MessageListJSON{
			ID:       v.ID,
			ThreadID: v.ThreadID,
		})
	}
	return c.JSON(http.StatusOK, jsonMessages)
}

// Returns Gmail message in JSON format.
func handleGmailGetMessage(c echo.Context) error {
	id := c.Param("ID")

	GmailClient, err := google.NewGmailClient(c)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	msg, err := GmailClient.GetMessage(id)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	return c.JSON(http.StatusOK, msg)
}

// Fetches message from Gmail by given ID as a parameter and writes it into SQLite Database in Storj.
// If there's no database yet - creates one.
func handleGmailMessageToStorj(c echo.Context) error {
	id := c.Param("ID")
	accesGrant, err := c.Cookie("storj_access_token")

	// FETCH THE EMAIL TO GOLANG STRUCT

	GmailClient, err := google.NewGmailClient(c)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	msg, err := GmailClient.GetMessage(id)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	msgToSave := storage.GmailMessageSQL{
		ID:      msg.ID,
		Date:    msg.Date,
		From:    msg.From,
		To:      msg.To,
		Subject: msg.Subject,
		Body:    msg.Body,
	}

	// SAVE ATTACHMENTS TO THE STORJ BUCKET AND WRITE THEIR NAMES TO STRUCT

	if len(msg.Attachments) > 0 {
		for _, att := range msg.Attachments {
			err = storj.UploadObject(context.Background(), accesGrant.Value, "gmail", att.FileName, att.Data)
			if err != nil {
				return c.String(http.StatusForbidden, err.Error())
			}
			msgToSave.Attachments = msgToSave.Attachments + "|" + att.FileName
		}
	}

	// CHECK IF EMAIL DATABASE ALREADY EXISTS AND DOWNLOAD IT, IF NOT - CREATE NEW ONE

	userCacheDBPath := "./cache/" + utils.CreateUserTempCacheFolder() + "/gmails.db"

	byteDB, err := storj.DownloadObject(context.Background(), accesGrant.Value, "gmail", "gmails.db")
	// Copy file from storj to local cache if everything's fine.
	// Skip error check, if there's error - we will check that and create new file
	if err == nil {
		dbFile, err := os.Create(userCacheDBPath)
		if err != nil {
			return c.String(http.StatusForbidden, err.Error())
		}
		_, err = dbFile.Write(byteDB)
		if err != nil {
			return c.String(http.StatusForbidden, err.Error())
		}
	}

	db, err := storage.ConnectToEmailDB()
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	// WRITE ALL EMAILS TO THE DATABASE LOCALLY

	err = db.WriteEmailToDB(&msgToSave)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	// DELETE OLD DB COPY FROM STORJ UPLOAD UP TO DATE DB FILE BACK TO STORJ AND DELETE IT FROM LOCAL CACHE

	// get db file data
	dbByte, err := os.ReadFile(userCacheDBPath)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	// delete old db copy from storj
	err = storj.DeleteObject(context.Background(), accesGrant.Value, "gmail", "gmails.db")
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	// upload file to storj
	err = storj.UploadObject(context.Background(), accesGrant.Value, "gmail", "gmails.db", dbByte)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	// delete from local cache copy of database
	err = os.Remove(userCacheDBPath)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	return c.String(http.StatusOK, "Email was successfully uploaded")
}

func handleGetGmailDBFromStorj(c echo.Context) error {
	accesGrant, err := c.Cookie("storj_access_token")

	// Copy file from storj to local cache if everything's fine.
	byteDB, err := storj.DownloadObject(context.Background(), accesGrant.Value, "gmail", "gmails.db")
	if err != nil {
		return c.String(http.StatusForbidden, "no emails saved in Storj database")
	}

	userCacheDBPath := "./cache/" + utils.CreateUserTempCacheFolder() + "/gmails.db"

	dbFile, err := os.Create(userCacheDBPath)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	_, err = dbFile.Write(byteDB)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}

	// delete db from cache after user get's it.
	defer os.Remove(userCacheDBPath)

	return c.Attachment(userCacheDBPath, "gmails.db")
}
