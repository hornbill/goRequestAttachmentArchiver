package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	_ "encoding/hex"
	"encoding/xml"
	_ "errors"
	"fmt"
	"github.com/cheggaaa/pb"
	"io"
	_ "io/ioutil"
	"math/rand"
	"net/http"
	_ "net/url"
	"os"
	_ "regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

func populateRequestsArray() {

	if configCall != "" {

		globalArrayRequests = append(globalArrayRequests, configCall)
		return

	} else if gStrCSVList != "" {

		file, err := os.Open(gStrCSVList)
		if err != nil {
			logger(4, err.Error(), true)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			clean := strings.Trim(scanner.Text(), "\"")
			if clean != "" {
				globalArrayRequests = append(globalArrayRequests, clean)
			}
		}

		if err := scanner.Err(); err != nil {
			logger(4, err.Error(), true)
		}

		return

	}

	localAPIKey := globalAPIKeys[0]
	espXmlmc := NewEspXmlmcSession(localAPIKey)

	//Determine Cut off date.
	t := time.Now()
	after := t.AddDate(0, 0, -1*7*configCutOff)
	cut_off_date := after.Format("2006-01-02") + " 00:00:00"
	logger(3, "Cut Off Date: "+cut_off_date, true)

	///////////////////
	espXmlmc.SetParam("application", "com.hornbill.servicemanager")
	espXmlmc.SetParam("queryName", "externalUtility_getOldRequestsWithAttachments")
	espXmlmc.OpenElement("queryParams")
	espXmlmc.SetParam("cut_off_date", cut_off_date)

	for _, serviceId := range importConf.Services {
		espXmlmc.SetParam("serviceId", strconv.Itoa(serviceId))
	}

	for _, status := range importConf.Statuses {
		espXmlmc.SetParam("status", status)
	}

	espXmlmc.CloseElement("queryParams")
	espXmlmc.OpenElement("queryOptions")
	espXmlmc.SetParam("resultType", "count")
	espXmlmc.CloseElement("queryOptions")

	RespBody, xmlmcErr := espXmlmc.Invoke("data", "queryExec")

	var JSONResp xmlmcCountResponse
	if xmlmcErr != nil {
		logger(4, "Unable to run Count Query "+fmt.Sprintf("%s", xmlmcErr), false)
	}
	err := xml.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		logger(4, "Unable to run Count Query "+fmt.Sprintf("%s", err), false)
	}
	if JSONResp.State.ErrorRet != "" {
		logger(4, "Unable to run Count Query "+JSONResp.State.ErrorRet, false)
	}

	//-- return Count
	count, errC := strconv.ParseUint(JSONResp.Params.RowData.Row[0].Count, 10, 32)
	//-- Check for Error
	if errC != nil {
		logger(4, "Unable to get Count for Count Query "+fmt.Sprintf("%s", err), false)
	} else {
		logger(3, "There are  "+fmt.Sprintf("%d", count)+" requests to be Processed", false)
	}
	///////////////////

	if count == 0 {
		return
	}

	var loopCount uint64

	bar := pb.StartNew(int(count))
	for loopCount < count {
		logger(1, "Loading Request Attachment List Offset: "+fmt.Sprintf("%d", loopCount)+"\n", false)

		espXmlmc.SetParam("application", "com.hornbill.servicemanager")
		espXmlmc.SetParam("queryName", "externalUtility_getOldRequestsWithAttachments")
		espXmlmc.OpenElement("queryParams")
		espXmlmc.SetParam("cut_off_date", cut_off_date)
		for _, serviceId := range importConf.Services {
			espXmlmc.SetParam("serviceId", strconv.Itoa(serviceId))
		}
		for _, status := range importConf.Statuses {
			espXmlmc.SetParam("status", status)
		}
		espXmlmc.SetParam("rowstart", strconv.FormatUint(loopCount, 10))
		espXmlmc.SetParam("limit", strconv.Itoa(configPageSize))
		espXmlmc.CloseElement("queryParams")
		espXmlmc.OpenElement("queryOptions")
		espXmlmc.SetParam("resultType", "references")
		espXmlmc.CloseElement("queryOptions")

		XMLAttachmentSearch, xmlmcErr := espXmlmc.Invoke("data", "queryExec")
		if xmlmcErr != nil {
			logger(6, "Unable to find Calls: "+fmt.Sprintf("%v", xmlmcErr), true)
			break
		}

		var xmlQuestionRespon structQueryResults
		qerr := xml.Unmarshal([]byte(XMLAttachmentSearch), &xmlQuestionRespon)

		if qerr != nil {
			fmt.Println("No Attachment Data Found")
			fmt.Println(qerr)
			break
		} else {
			if xmlQuestionRespon.MethodResult == "fail" {
				fmt.Println(xmlQuestionRespon.State.ErrorRet)
				break
			}
			intResponseSize := len(xmlQuestionRespon.Params.RowData.Row)
			logger(3, "RowResults: "+strconv.Itoa(intResponseSize), false)

			for i := 0; i < intResponseSize; i++ {
				globalArrayRequests = append(globalArrayRequests, xmlQuestionRespon.Params.RowData.Row[i].HOwnerKey)
			}
		}

		// Add 100
		loopCount += uint64(configPageSize)
		bar.Add(len(xmlQuestionRespon.Params.RowData.Row))
		//-- Check for empty result set
		if len(xmlQuestionRespon.Params.RowData.Row) == 0 {
			break
		}

	}
	logger(3, "Found "+strconv.Itoa(len(globalArrayRequests))+" Calls with attachments", false)
	bar.FinishPrint("Requests Loaded \n")
}

func checkAPIKeys() bool {

	logger(3, "Checking API Keys", false)
	intAPIKeysLength := len(importConf.APIKeys)

	for i := 0; i < intAPIKeysLength; i++ {

		logger(3, "Checking API Key : "+importConf.APIKeys[i], false)

		espXmlmc := NewEspXmlmcSession(importConf.APIKeys[i])
		espXmlmc.SetParam("stage", "1")
		strAPIResult, xmlmcErr := espXmlmc.Invoke("system", "pingCheck")
		if xmlmcErr != nil {
			logger(4, "Failed PingCheck for : "+importConf.APIKeys[i], false)
		} else {
			var xmlQuestionRespon structQueryResults
			qerr := xml.Unmarshal([]byte(strAPIResult), &xmlQuestionRespon)
			if qerr != nil || xmlQuestionRespon.MethodResult == "fail" {
				//fmt.Println(strAPIResult)
				//fmt.Println(xmlQuestionRespon.State.ErrorRet)
				logger(5, "Found "+importConf.APIKeys[i]+" to be an invalid API key", true)
			} else {
				globalAPIKeys = append(globalAPIKeys, importConf.APIKeys[i])
			}
		}
	}

	logger(3, "Found "+strconv.Itoa(len(globalAPIKeys))+" valid API Keys", true)

	return len(globalAPIKeys) > 0
}

func pickOffRequestArray() (bool, string) {
	boolReturn := false
	stringLastItem := ""

	if len(globalArrayRequests) > 0 {
		boolReturn = true
		mutex.Lock()
		stringLastItem = globalArrayRequests[len(globalArrayRequests)-1]
		globalArrayRequests[len(globalArrayRequests)-1] = ""
		globalArrayRequests = globalArrayRequests[:len(globalArrayRequests)-1]
		mutex.Unlock()
		globalArrayBars[0].Increment()
	}
	boolReturn = !(stringLastItem == "")
	return boolReturn, stringLastItem
}

func addToProcessedArray(processedRequestID string) {
	mutex.Lock()
	globalArrayProcessed = append(globalArrayProcessed, processedRequestID)
	mutex.Unlock()
}

func setOutputFolder() {
	localFolder := ""

	if importConf.AttachmentFolder != "" {
		localFolder = importConf.AttachmentFolder
	}
	if configOutputFolder != "" {
		localFolder = configOutputFolder
	}

	logger(2, "Checking "+localFolder, false)
	if src, err := os.Stat(localFolder); !os.IsNotExist(err) {
		//folder/file exists
		if !src.IsDir() {
			//not a directory
			logger(5, localFolder+" is not a folder.", true)
		} else {
			if src.Mode().Perm()&(1<<(uint(7))) == 0 {
				logger(5, "Write permission not set on this folder.", true)
			} else {
				globalAttachmentLocation = localFolder
			}
		}
	} else {
		logger(5, localFolder+" does not exist, trying to create folder", true)
		err := os.Mkdir(localFolder, 0777)
		if err == nil {
			//folder creation successful, so use created folder
			globalAttachmentLocation = localFolder
		}

	}

	if globalAttachmentLocation == "" {
		logger(2, "Using current folder for attachments", false)
		globalAttachmentLocation = "."
	}

	logger(2, "Using: "+globalAttachmentLocation, false)

}

func processCalls(threadId int) {

	localAPIKey := globalAPIKeys[threadId]
	localLink := NewEspXmlmcSession(localAPIKey)

	localBar := globalArrayBars[threadId+1]

	for {
		boolIDExists, requestID := pickOffRequestArray()

		//fmt.Println(requestID)
		if !boolIDExists {
			logger(3, "Finished Thread "+strconv.Itoa(threadId+1), false)
			break
		} else {
			logger(3, "Processing: "+requestID, false)

			localLink.SetParam("application", "com.hornbill.servicemanager")
			localLink.SetParam("entity", "Requests")
			localLink.SetParam("keyValue", requestID)

			XMLAttachmentSearch, xmlmcErr := localLink.Invoke("data", "entityAttachBrowse")
			if xmlmcErr != nil {
				logger(4, "Unable to find attachments for: "+requestID+" - "+fmt.Sprintf("%v", xmlmcErr), false)
				continue
			}

			var xmlQuestionRespon structAttachmentsResults
			//fmt.Println(XMLAttachmentSearch)
			qerr := xml.Unmarshal([]byte(XMLAttachmentSearch), &xmlQuestionRespon)

			if qerr != nil {
				fmt.Println("No Attachment Data Found for " + requestID)
				fmt.Println(qerr)
			} else {
				intCountDownloads := len(xmlQuestionRespon.Params.File)
				if intCountDownloads == 0 {
					logger(3, "No downloads found for: "+requestID, false)
					continue
					//return
				}
				logger(3, strconv.Itoa(intCountDownloads)+" downloads found for: "+requestID, false)

				localBar.Finish()
				localBar.Reset(intCountDownloads)

				var downloadedFiles []string

				strFileList := ""

				if configDoNotArchiveFiles {

					strFileList = "Files removed on " + globalNiceTime + ":\r\n"

					for i := 0; i < intCountDownloads; i++ {
						strFileName := xmlQuestionRespon.Params.File[i].File.HFileName
						strFileList += "\r\n" + strFileName
						downloadedFiles = append(downloadedFiles, strFileName)
						localBar.Increment()
					}

				} else {

					newZipFile, err := os.Create(globalAttachmentLocation + string(os.PathSeparator) + requestID + "_" + globalTimeNow + ".zip")
					if err != nil {
						logger(4, "Unable to open .ZIP file for: "+requestID+" - "+fmt.Sprintf("%v", err), false)
						continue
					}

					zipWriter := zip.NewWriter(newZipFile)

					strFileList = "Files archived on " + globalNiceTime + ":\r\n"

					for i := 0; i < intCountDownloads; i++ {

						//20200910 strContentLocation := xmlQuestionRespon.Params.RowData.Row[i].HContentLocation
						strFileName := xmlQuestionRespon.Params.File[i].File.HFileName
						strAccessToken := xmlQuestionRespon.Params.File[i].AccessToken
						//fmt.Println(strContentLocation)
						var emptyCatch []byte

						time.Sleep(time.Millisecond * time.Duration(rand.Intn(2000))) //think this might be necessary

						strDAVurl := localLink.DavEndpoint
						strDAVurl = strDAVurl + "secure-content/download/" + strAccessToken
						logger(1, "GETting: "+strFileName, false)

						putbody := bytes.NewReader(emptyCatch)
						req, Perr := http.NewRequest("GET", strDAVurl, putbody)
						if Perr != nil {
							logger(3, "GET set-up issue", false)
							continue
						}
						req.Header.Add("Authorization", "ESP-APIKEY "+localAPIKey) //APIKey)
						req.Header.Set("User-Agent", "Go-http-client/1.1")
						response, Perr := client.Do(req)
						if Perr != nil {
							logger(3, "GET connection issue: "+fmt.Sprintf("%v", http.StatusInternalServerError), false)
							continue
						}

						//Sanitizing filename - for use in .zip
						strFileName = strings.ReplaceAll(strFileName, "\n", "") // as NewLine characters appear to have creeped into the file name (my guess: email header not being sanitized)
						strFileName = strings.ReplaceAll(strFileName, "\r", "") // better safe than sorry
						strFileName = strings.ReplaceAll(strFileName, "*", "")
						strFileName = strings.ReplaceAll(strFileName, "?", "")
						strFileName = strings.ReplaceAll(strFileName, "\\", "_")
						strFileName = strings.ReplaceAll(strFileName, "/", "_")
						strFileName = strings.ReplaceAll(strFileName, ":", "_")
						strFileName = strings.ReplaceAll(strFileName, "|", "_")
						strFileName = strings.ReplaceAll(strFileName, ">", "_")
						strFileName = strings.ReplaceAll(strFileName, "<", "_")

						//logger(3, fmt.Sprintf("Received data: %d bytes", response.ContentLength), false) //- content length was -1 (known Go issue)

						if response.StatusCode == 200 {
							header := &zip.FileHeader{
								//Name:   xmlQuestionRespon.Params.File[i].File.HFileName,
								Name:   strFileName,
								Method: zip.Deflate,
							}

							writer, err := zipWriter.CreateHeader(header)
							if err != nil {
								logger(1, "Zip Header Error: "+fmt.Sprintf("%v", err), false)
								response.Body.Close()
								continue
							} else {
								_, err = io.Copy(writer, response.Body)
								if err != nil {
									logger(1, "io.Copy Error: "+fmt.Sprintf("%v", err), false)
									response.Body.Close()
									continue
								}
							}

							strFileList += "\r\n" + xmlQuestionRespon.Params.File[i].File.HFileName
							// yeah do NOT use sanitized filename here!
							downloadedFiles = append(downloadedFiles, xmlQuestionRespon.Params.File[i].File.HFileName)

						} else {
							logger(1, "Unsuccesful Download: "+fmt.Sprintf("%v", response.StatusCode), false)
						}

						err = response.Body.Close()
						if err != nil {
							logger(1, "Body Close Error: "+fmt.Sprintf("%v", err), false)
						}
						localBar.Increment()

					}

					err = zipWriter.Close()
					if err != nil {
						logger(1, "zipWriter Close Error: "+fmt.Sprintf("%v", err), false)
						downloadedFiles = nil // better ensure we are not removing anything
					}
					err = newZipFile.Close()
					if err != nil {
						logger(1, "newZipFile Close Error: "+fmt.Sprintf("%v", err), false)
						downloadedFiles = nil // better ensure we are not removing anything
					}
				}

				iDownloadedFiles := len(downloadedFiles)

				if configDoNotArchiveFiles {
					logger(1, "Items lined up for removal: "+fmt.Sprintf("%d", iDownloadedFiles), false)
				} else {
					logger(1, "Succesful Downloads: "+fmt.Sprintf("%d", iDownloadedFiles), false)
				}

				if !(configDryRun) && iDownloadedFiles > 0 {
					for i := 0; i < iDownloadedFiles; i++ {
						logger(3, "Removal of "+downloadedFiles[i]+" from "+requestID, false)
						//we've got the file, so now let's remove from source:
						localLink.SetParam("application", "com.hornbill.servicemanager")
						localLink.SetParam("entity", "Requests")
						localLink.SetParam("keyValue", requestID)
						localLink.SetParam("filePath", downloadedFiles[i])
						_, xmlmcErr := localLink.Invoke("data", "entityAttachRemove")
						if xmlmcErr != nil {
							logger(4, "Unable to remove attachment: "+downloadedFiles[i]+" from "+requestID, false)
							//need to decide what to do if unable to remove attachment - it might be because it didn't exist in the first place
						} else {
							logger(1, "Processed: "+downloadedFiles[i]+" for "+requestID, false)
						}
					}
					//update call with strFileList
					if configRequestUpdate {
						localLink.SetParam("requestId", requestID)
						localLink.SetParam("content", strFileList)
						localLink.SetParam("visibility", "colleague")
						localLink.SetParam("activityType", "Archiver")
						localLink.SetParam("skipBpm", "true")
						_, xmlmcErr := localLink.Invoke("apps/com.hornbill.servicemanager/Requests", "updateReqTimeline")
						if xmlmcErr != nil {
							logger(4, "Unable to Update "+requestID, false)
							//need to decide what to do if unable to remove attachment - it might be because it didn't exist in the first place
						}
					}
				} else {
					logger(3, fmt.Sprintf("Skipping removal of %d files from %s", iDownloadedFiles, requestID), false)
				}

				addToProcessedArray(requestID)

			}

		}
	}

	localBar.Finish()

}

func main() {
	startTime = time.Now()
	//-- Start Time for Log File
	globalTimeNow = time.Now().Format(time.RFC3339)

	globalNiceTime = globalTimeNow[:16]
	globalNiceTime = strings.Replace(globalNiceTime, "T", " ", 1)

	globalTimeNow = strings.Replace(globalTimeNow, ":", "-", -1)
	localLogFileName += globalTimeNow
	localLogFileName += ".log"

	parseFlags()

	//-- Output to CLI and Log
	logger(1, "---- Hornbill Request Attachment Download and Removal Utility v"+fmt.Sprintf("%v", version)+" ----", false)
	logger(1, "Flag - Config File "+configFileName, false)
	logger(1, "Flag - Dry Run "+fmt.Sprintf("%v", configDryRun), false)

	//-- Load Configuration File Into Struct
	boolConfLoaded := false
	importConf, boolConfLoaded = loadConfig()
	if !boolConfLoaded {
		logger(4, "Unable to load config, process closing.", true)
		return
	}
	if (configCall == "" && gStrCSVList != "") || (configCall != "" && gStrCSVList == "") {
		if !(configOverride) && configCutOff < globalUltimateCutOff {
			logger(4, "The cut off date is too short (must be >= 12 (weeks)), process closing.", true)
			return
		}
	}
	if !(checkAPIKeys()) {
		logger(4, "No valid API keys.", true)
		return
	}

	globalMaxRoutines = len(globalAPIKeys)
	if globalMaxRoutines < 1 || globalMaxRoutines > 10 {
		logger(5, "The maximum allowed workers is between 1 and 10 (inclusive).", true)
		logger(4, "You have included "+strconv.Itoa(globalMaxRoutines)+" API keys. Please try again, with a valid number of keys.", true)
		return
	}

	setOutputFolder()

	populateRequestsArray()

	if len(globalArrayRequests) > 0 {

		globalBarRequests = pb.New(len(globalArrayRequests)).Prefix("Overall :")

		globalArrayBars = append(globalArrayBars, globalBarRequests)

		amount_per_bar := len(globalArrayRequests) / globalMaxRoutines
		if amount_per_bar > 0 && globalMaxRoutines > 1 {
			logger(1, "Spawning multiple processes", false)

			var wg sync.WaitGroup
			wg.Add(globalMaxRoutines)

			for i := 0; i < globalMaxRoutines; i++ {
				ppp := pb.New(amount_per_bar).Prefix("Thread " + strconv.Itoa(i+1) + ":")
				ppp.ShowTimeLeft = false
				ppp.ShowCounters = false
				ppp.ShowFinalTime = false
				globalArrayBars = append(globalArrayBars, ppp)
			}
			pool, err := pb.StartPool(globalArrayBars...)

			if err != nil {
				panic(err)
			}

			for i := 0; i < globalMaxRoutines; i++ {
				go func(i int) {
					defer wg.Done()
					processCalls(i)
				}(i)
			}
			wg.Wait()

			pool.Stop()

		} else {
			logger(1, "Just a single process", false)
			//presumably == 0 or just a single thread, so just need a single total bar.
			ppp := pb.New(1).Prefix("Main Thread :")
			//			pool.Add(ppp)
			globalArrayBars = append(globalArrayBars, ppp)
			pool, err := pb.StartPool(globalArrayBars...)

			if err != nil {
				panic(err)
			}
			processCalls(0)
			globalArrayBars[0].Finish()
			pool.Stop()

		}
	} else {
		fmt.Println("No downloads found")
	}

	//-- End output
	//logger(3, "Requests Logged: "+fmt.Sprintf("%d", counters.created), true)
	//-- Show Time Takens
	endTime = time.Since(startTime)
	logger(3, "Time Taken: "+fmt.Sprintf("%v", endTime), true)
	logger(1, "---- Hornbill Request Attachment Download and Removal Complete ---- ", false)

}
