package main

import (
	"github.com/cheggaaa/pb"
	_ "github.com/hornbill/goApiLib"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	version              = "1.4.0"
	applicationName      = "Hornbill Request Attachment Archiver"
	appName              = "goRequestAttachmentArchiver"
	execName             = "goRequestAttachmentArchiver"
	appServiceManager    = "com.hornbill.servicemanager"
	globalUltimateCutOff = 12
	globalDefaultCutOff  = 0
)

var (
	localLogFileName         = "RAA_"
	espLogFileName           = "RequestAttachmentArchiver"
	configFileName           string
	configDryRun             bool
	configOverride           bool
	configRequestUpdate      bool
	configDoNotArchiveFiles  bool
	configOutputFolder       string
	configCutOff             int
	configCall               = ""
	gStrCSVList              = ""
	configPageSize           = 100
	globalMaxRoutines        int
	globalAPIKeys            []string
	globalTimeNow            string
	globalNiceTime           string
	globalAttachmentLocation = ""
	importConf               importConfStruct
	startTime                time.Time
	endTime                  time.Duration
	wg                       sync.WaitGroup
	client                   = http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   600 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   1,
			Proxy:                 http.ProxyFromEnvironment,
		},
		//Timeout: time.Duration(120 * time.Second),
	}

	mutex                = &sync.Mutex{}
	globalArrayRequests  []string
	globalArrayProcessed []string
	globalBarRequests    *pb.ProgressBar
	globalArrayBars      []*pb.ProgressBar
	//globalArrayLinks     []*apiLib.XmlmcInstStruct
)

// ----- Config Data Structs
type importConfStruct struct {
	InstanceID       string
	APIKeys          []string
	AttachmentFolder string
	Services         []int
	Statuses         []string
}

type stateStruct struct {
	Code     string `xml:"code"`
	ErrorRet string `xml:"error"`
}
type HBResults struct {
	HID              int    `xml:"h_pk_id"`
	HReqID           string `xml:"h_request_id"`
	HContentLocation string `xml:"h_contentlocation"`
	HFileName        string `xml:"h_filename"`
	HSize            int    `xml:"h_size"`
	HTimeStamp       string `xml:"h_timestamp"`
	HVisibility      string `xml:"h_visibility"`
	HCount           string `xml:"h_count"`
	HOwnerKey        string `xml:"h_owner_key"`
}

type fileResults struct {
	File struct {
		HFileID     int    `xml:"fileId"`
		HFileSource string `xml:"fileSource"`
		HFileName   string `xml:"fileName"`
		HSize       int    `xml:"fileSize"`
		HTimeStamp  string `xml:"timeStamp"`
	} `xml:"fileInfo"`
	AccessToken string `xml:"accessToken"`
}

type structQueryResults struct {
	MethodResult string `xml:"status,attr"`
	Params       struct {
		RowData struct {
			Row []HBResults `xml:"row"`
		} `xml:"rowData"`
	} `xml:"params"`
	State stateStruct `xml:"state"`
}
type xmlmcCountResponse struct {
	Params struct {
		RowData struct {
			Row []struct {
				Count string `xml:"h_count"`
			} `xml:"row"`
		} `xml:"rowData"`
	} `xml:"params"`
	State stateStruct `xml:"state"`
}

type structAttachmentsResults struct {
	MethodResult string `xml:"status,attr"`
	Params       struct {
		File []fileResults `xml:"file"`
	} `xml:"params"`
	State stateStruct `xml:"state"`
}
