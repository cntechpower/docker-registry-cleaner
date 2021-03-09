package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"log"

	"github.com/docker/distribution/registry/client"
)

type TagList struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}
type imageInfo struct {
	Arch          string `json:"architecture"`
	Created       string `json:"created"`
	Tag           string `json:"-"`
	DigestForInfo string `json:"-"`
	DigestForDel  string `json:"-"`
}

type manifests struct {
	Digest    string           `json:"-"`
	Config    *manifestsConfig `json:"config"`
	MediaType string           `json:"mediaType"`
}

type manifestsConfig struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
}

var registryUrl string
var tagsKeep int
var repoBuffer int
var dryRun bool
var listAll bool

func init() {
	flag.StringVar(&registryUrl, "registry-url", "http://127.0.0.1:5000", "docker registry url")
	flag.IntVar(&tagsKeep, "tags-keep", 5, "tags to keep")
	flag.IntVar(&repoBuffer, "repos-buffer", 9999, "repos buffer, should not small than total repo count")
	flag.BoolVar(&dryRun, "dry-run", true, "only show which tag will be delete, but not actual delete")
	flag.BoolVar(&listAll, "list-all", false, "show all images tags")
}

func main() {
	flag.Parse()
	ctx := context.Background()
	reg, err := client.NewRegistry(registryUrl, nil)
	if err != nil {
		log.Printf("connect to docker registry error: %v", err)
		return
	}
	repos := make([]string, repoBuffer)
	n, err := reg.Repositories(ctx, repos, "")
	if err != nil && err.Error() != "EOF" {
		log.Printf("list repos error: %v", err)
		return
	}
	if n >= repoBuffer {
		log.Printf("repos count may bigger than repos-buffer, please try to increase repos-buffer")
		return
	}

	imagesShouldClean := make(map[string] /*repo*/ []*imageInfo, 0)
	for i := 0; i < n; i++ {
		r := repos[i]
		tags, err := getImageTags(registryUrl, r)
		if err != nil {
			log.Printf("get tags for %v error: %v", r, err)
			continue
		}
		//do not delete if tags smaller than tagsKeep
		if !listAll && len(tags.Tags) <= tagsKeep {
			continue
		}
		images := make([]*imageInfo, 0, len(tags.Tags))
		for _, tag := range tags.Tags {
			mani, err := getImageManifests(registryUrl, r, tag)
			if err != nil {
				log.Printf("get image manifests for %v:%v error: %v", r, tag, err)
				continue
			}
			info, err := getImageInfo(registryUrl, r, mani.Config.Digest)
			if err != nil {
				log.Printf("get image info for %v:%v error: %v", r, tag, err)
				continue
			}
			info.Tag = tag
			info.DigestForInfo = mani.Config.Digest
			info.DigestForDel = mani.Digest
			images = append(images, info)
		}
		imagesShouldClean[r] = images
	}
	if listAll {
		for imageName, infos := range imagesShouldClean {
			for _, info := range infos {
				log.Printf("%v:%v", imageName, info.Tag)
			}
		}
	}
	if len(imagesShouldClean) == 0 {
		log.Printf("no images need to deleted")
	}
	//judge which image:tag should be deleted
	for imageName, info := range imagesShouldClean {
		//sort by created date
		sort.Slice(info, func(i, j int) bool {
			ti, err := time.Parse(time.RFC3339Nano, info[i].Created)
			if err != nil {
				panic(err)
			}
			tj, err := time.Parse(time.RFC3339Nano, info[j].Created)
			if err != nil {
				panic(err)
			}
			return ti.Unix() < tj.Unix()
		})
		for i := 0; i < len(info)-tagsKeep; i++ {
			log.Printf("will delete %v:%v, create time %v", imageName, info[i].Tag, info[i].Created)
			if !dryRun {
				if err := delImage(registryUrl, imageName, info[i].DigestForDel); err != nil {
					log.Printf("delete %v:%v error: %v", imageName, info[i].Tag, err)
				} else {
					log.Printf("delete %v:%v suceess", imageName, info[i].Tag)
				}
			}
		}
	}
}

func getImageInfo(baseUrl, image, tag string) (info *imageInfo, err error) {
	info = &imageInfo{}
	url := fmt.Sprintf(`%v/v2/%v/blobs/%v`, baseUrl, image, tag)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(bs, info)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected http code %v", resp.StatusCode)
	}
	return
}

func getImageTags(baseUrl, image string) (info *TagList, err error) {
	info = &TagList{
		Name: "",
		Tags: make([]string, 0),
	}
	url := fmt.Sprintf(`%v/v2/%v/tags/list`, baseUrl, image)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(bs, &info)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("GET %v, got unexpected http code %v", url, resp.StatusCode)
	}
	return
}

func getImageManifests(baseUrl, image, tag string) (info *manifests, err error) {
	info = &manifests{}
	url := fmt.Sprintf(`%v/v2/%v/manifests/%v`, baseUrl, image, tag)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	//https://docs.docker.com/registry/spec/api/#deleting-an-image
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(bs, &info)
	if err != nil {
		return
	}
	info.Digest = resp.Header.Get("Docker-Content-Digest")
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("GET %v, got unexpected http code %v", url, resp.StatusCode)
	}
	return
}

func delImage(baseUrl, image, digest string) (err error) {
	url := fmt.Sprintf(`%v/v2/%v/manifests/%v`, baseUrl, image, digest)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusAccepted {
		err = fmt.Errorf("DELETE %v, got unexpected http code %v", url, resp.StatusCode)
	}
	return
}
