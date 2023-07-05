package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/html"
)

const (
	baseURL = "https://mcdc.missouri.edu"
)

func main() {
	latitude, longitude, radii, outputFilePrefix := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	err := downloadCAPSData(latitude, longitude, radii, outputFilePrefix)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func downloadCAPSData(latitude, longitude, radii, outputFilePrefix string) error {
	acsResponse, err := submitCAPSACSRequest(latitude, longitude, radii)
	if err != nil {
		return err
	}
	acsCSVPath := extractGeneratedCSVPath(acsResponse)
	if acsCSVPath == "" {
		return fmt.Errorf("ACS response did not contain expected temporary CSV link")
	}
	censusResponse, err := submitCAPS2010Request(latitude, longitude, radii)
	if err != nil {
		return err
	}
	censusCSVPath := extractGeneratedCSVPath(censusResponse)
	if acsCSVPath == "" {
		return fmt.Errorf("census response did not contain expected temporary CSV link")
	}
	err = downloadCSV(acsCSVPath, outputFilePrefix+"_ACS.csv")
	if err != nil {
		return err
	}
	err = downloadCSV(censusCSVPath, outputFilePrefix+"_2010.csv")
	if err != nil {
		return err
	}
	return nil
}

func submitCAPSACSRequest(latitude, longitude, radii string) (*html.Node, error) {
	resource := "/cgi-bin/broker"
	params := url.Values{}
	params.Add("_PROGRAM", "apps.capsACS.sas")
	params.Add("_SERVICE", "MCDC_long")
	params.Add("latitude", latitude)
	params.Add("longitude", longitude)
	params.Add("radii", radii)
	params.Add("sitename", "")
	params.Add("dprofile", "on")
	params.Add("eprofile", "on")
	params.Add("sprofile", "on")
	params.Add("hprofile", "on")
	params.Add("units", " ")

	u, _ := url.ParseRequestURI(baseURL)
	u.Path = resource
	u.RawQuery = params.Encode()
	urlStr := fmt.Sprintf("%v", u)

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("submitting CAPS ACS request: %w", err)
	}
	defer resp.Body.Close()

	return html.Parse(resp.Body)
}

func submitCAPS2010Request(latitude, longitude, radii string) (*html.Node, error) {
	resource := "/cgi-bin/broker"
	params := url.Values{}
	params.Add("_PROGRAM", "apps.caps2010.sas")
	params.Add("debug", "")
	params.Add("latitude", latitude)
	params.Add("longitude", longitude)
	params.Add("radii", radii)
	params.Add("sitename", "")
	params.Add("units", " ")

	u, _ := url.ParseRequestURI(baseURL)
	u.Path = resource
	u.RawQuery = params.Encode()
	urlStr := fmt.Sprintf("%v", u)

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("submitting CAPS 2010 request: %w", err)
	}
	defer resp.Body.Close()

	return html.Parse(resp.Body)
}

func extractGeneratedCSVPath(doc *html.Node) string {
	var tempCSVPath string
	var link func(*html.Node)
	link = func(n *html.Node) {
		if n.Type == html.TextNode && n.Data == "CSV file of aggregated data" {
			for _, a := range n.Parent.Attr {
				if a.Key == "href" {
					// get link href for temporary generated CSV file
					tempCSVPath = a.Val
				}
			}
		}

		// traverses the HTML of the webpage from the first child node
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			link(c)
		}
	}
	link(doc)
	return tempCSVPath
}

func downloadCSV(tempCSVPath, fileName string) error {
	u, _ := url.ParseRequestURI(baseURL)
	u.Path = tempCSVPath
	urlStr := fmt.Sprintf("%v", u)
	resp, err := http.Get(urlStr)
	if err != nil {
		return fmt.Errorf("downloading CAPS CSV data: %w", err)
	}
	defer resp.Body.Close()

	// Create blank file
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("copying data to output file: %w", err)
	}

	fmt.Printf("Downloaded a file %s with size %d bytes.\n", fileName, size)
	return nil
}
