/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package donut

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	. "github.com/minio/check"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

// create a dummy TestNodeDiskMap
func createTestNodeDiskMap(p string) map[string][]string {
	nodes := make(map[string][]string)
	nodes["localhost"] = make([]string, 16)
	for i := 0; i < len(nodes["localhost"]); i++ {
		diskPath := filepath.Join(p, strconv.Itoa(i))
		if _, err := os.Stat(diskPath); err != nil {
			if os.IsNotExist(err) {
				os.MkdirAll(diskPath, 0700)
			}
		}
		nodes["localhost"][i] = diskPath
	}
	return nodes
}

// test empty donut
func (s *MySuite) TestEmptyDonut(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut, err := NewDonut("test", createTestNodeDiskMap(root))
	c.Assert(err, IsNil)

	// check donut is empty
	metadata, err := donut.ListBuckets()
	c.Assert(err, IsNil)
	c.Assert(len(metadata), Equals, 0)
}

// test make bucket without name
func (s *MySuite) TestBucketWithoutNameFails(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut, err := NewDonut("test", createTestNodeDiskMap(root))
	c.Assert(err, IsNil)
	// fail to create new bucket without a name
	err = donut.MakeBucket("", "private")
	c.Assert(err, Not(IsNil))

	err = donut.MakeBucket(" ", "private")
	c.Assert(err, Not(IsNil))
}

// test empty bucket
func (s *MySuite) TestEmptyBucket(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut, err := NewDonut("test", createTestNodeDiskMap(root))
	c.Assert(err, IsNil)

	c.Assert(donut.MakeBucket("foo", BucketACL("private")), IsNil)
	// check if bucket is empty
	listObjects, err := donut.ListObjects("foo", "", "", "", 1)
	c.Assert(err, IsNil)
	c.Assert(len(listObjects.Objects), Equals, 0)
	c.Assert(listObjects.CommonPrefixes, DeepEquals, []string{})
	c.Assert(listObjects.IsTruncated, Equals, false)
}

// test bucket list
func (s *MySuite) TestMakeBucketAndList(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut, err := NewDonut("test", createTestNodeDiskMap(root))
	c.Assert(err, IsNil)
	// create bucket
	err = donut.MakeBucket("foo", BucketACL("private"))
	c.Assert(err, IsNil)

	// check bucket exists
	buckets, err := donut.ListBuckets()
	c.Assert(err, IsNil)
	c.Assert(len(buckets), Equals, 1)
	c.Assert(buckets["foo"].ACL, Equals, BucketACL("private"))
}

// test re-create bucket
func (s *MySuite) TestMakeBucketWithSameNameFails(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut, err := NewDonut("test", createTestNodeDiskMap(root))
	c.Assert(err, IsNil)
	err = donut.MakeBucket("foo", BucketACL("private"))
	c.Assert(err, IsNil)

	err = donut.MakeBucket("foo", BucketACL("private"))
	c.Assert(err, Not(IsNil))
}

// test make multiple buckets
func (s *MySuite) TestCreateMultipleBucketsAndList(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut, err := NewDonut("test", createTestNodeDiskMap(root))
	c.Assert(err, IsNil)
	// add a second bucket
	err = donut.MakeBucket("foo", BucketACL("private"))
	c.Assert(err, IsNil)

	err = donut.MakeBucket("bar", BucketACL("private"))
	c.Assert(err, IsNil)

	buckets, err := donut.ListBuckets()
	c.Assert(err, IsNil)

	_, ok := buckets["foo"]
	c.Assert(ok, Equals, true)
	_, ok = buckets["bar"]
	c.Assert(ok, Equals, true)

	err = donut.MakeBucket("foobar", BucketACL("private"))
	c.Assert(err, IsNil)

	buckets, err = donut.ListBuckets()
	c.Assert(err, IsNil)
	_, ok = buckets["foobar"]
	c.Assert(ok, Equals, true)
}

// test object create without bucket
func (s *MySuite) TestNewObjectFailsWithoutBucket(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut, err := NewDonut("test", createTestNodeDiskMap(root))
	c.Assert(err, IsNil)
	_, err = donut.PutObject("foo", "obj", "", nil, nil)
	c.Assert(err, Not(IsNil))
}

// test create object metadata
func (s *MySuite) TestNewObjectMetadata(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut, err := NewDonut("test", createTestNodeDiskMap(root))
	c.Assert(err, IsNil)

	metadata := make(map[string]string)
	metadata["contentType"] = "application/json"
	metadata["foo"] = "value1"
	metadata["hello"] = "world"

	data := "Hello World"
	hasher := md5.New()
	hasher.Write([]byte(data))
	expectedMd5Sum := hex.EncodeToString(hasher.Sum(nil))
	reader := ioutil.NopCloser(bytes.NewReader([]byte(data)))
	metadata["contentLength"] = strconv.Itoa(len(data))

	err = donut.MakeBucket("foo", "private")
	c.Assert(err, IsNil)

	objectMetadata, err := donut.PutObject("foo", "obj", expectedMd5Sum, reader, metadata)
	c.Assert(err, IsNil)
	c.Assert(objectMetadata.MD5Sum, Equals, expectedMd5Sum)
	c.Assert(objectMetadata.Metadata["contentType"], Equals, metadata["contentType"])
	c.Assert(objectMetadata.Metadata["foo"], Equals, metadata["foo"])
	c.Assert(objectMetadata.Metadata["hello"], Equals, metadata["hello"])
}

// test create object fails without name
func (s *MySuite) TestNewObjectFailsWithEmptyName(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut, err := NewDonut("test", createTestNodeDiskMap(root))
	c.Assert(err, IsNil)

	_, err = donut.PutObject("foo", "", "", nil, nil)
	c.Assert(err, Not(IsNil))
}

// test create object
func (s *MySuite) TestNewObjectCanBeWritten(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut, err := NewDonut("test", createTestNodeDiskMap(root))
	c.Assert(err, IsNil)

	err = donut.MakeBucket("foo", "private")
	c.Assert(err, IsNil)

	metadata := make(map[string]string)
	metadata["contentType"] = "application/octet-stream"
	data := "Hello World"

	hasher := md5.New()
	hasher.Write([]byte(data))
	expectedMd5Sum := hex.EncodeToString(hasher.Sum(nil))
	reader := ioutil.NopCloser(bytes.NewReader([]byte(data)))
	metadata["contentLength"] = strconv.Itoa(len(data))

	actualMetadata, err := donut.PutObject("foo", "obj", expectedMd5Sum, reader, metadata)
	c.Assert(err, IsNil)
	c.Assert(actualMetadata.MD5Sum, Equals, expectedMd5Sum)

	reader, size, err := donut.GetObject("foo", "obj")
	c.Assert(err, IsNil)
	c.Assert(size, Equals, int64(len(data)))

	var actualData bytes.Buffer
	_, err = io.Copy(&actualData, reader)
	c.Assert(err, IsNil)
	c.Assert(actualData.Bytes(), DeepEquals, []byte(data))

	actualMetadata, err = donut.GetObjectMetadata("foo", "obj")
	c.Assert(err, IsNil)
	c.Assert(expectedMd5Sum, Equals, actualMetadata.MD5Sum)
	c.Assert(int64(len(data)), Equals, actualMetadata.Size)
	c.Assert("1.0.0", Equals, actualMetadata.Version)
}

// test list objects
func (s *MySuite) TestMultipleNewObjects(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut, err := NewDonut("test", createTestNodeDiskMap(root))
	c.Assert(err, IsNil)

	c.Assert(donut.MakeBucket("foo", BucketACL("private")), IsNil)

	one := ioutil.NopCloser(bytes.NewReader([]byte("one")))
	metadata := make(map[string]string)
	metadata["contentLength"] = strconv.Itoa(len("one"))

	_, err = donut.PutObject("foo", "obj1", "", one, metadata)
	c.Assert(err, IsNil)

	two := ioutil.NopCloser(bytes.NewReader([]byte("two")))

	metadata["contentLength"] = strconv.Itoa(len("two"))
	_, err = donut.PutObject("foo", "obj2", "", two, metadata)
	c.Assert(err, IsNil)

	obj1, size, err := donut.GetObject("foo", "obj1")
	c.Assert(err, IsNil)
	c.Assert(size, Equals, int64(len([]byte("one"))))

	var readerBuffer1 bytes.Buffer
	_, err = io.CopyN(&readerBuffer1, obj1, size)
	c.Assert(err, IsNil)
	c.Assert(readerBuffer1.Bytes(), DeepEquals, []byte("one"))

	obj2, size, err := donut.GetObject("foo", "obj2")
	c.Assert(err, IsNil)
	c.Assert(size, Equals, int64(len([]byte("two"))))

	var readerBuffer2 bytes.Buffer
	_, err = io.CopyN(&readerBuffer2, obj2, size)
	c.Assert(err, IsNil)
	c.Assert(readerBuffer2.Bytes(), DeepEquals, []byte("two"))

	/// test list of objects

	// test list objects with prefix and delimiter
	listObjects, err := donut.ListObjects("foo", "o", "", "1", 10)
	c.Assert(err, IsNil)
	c.Assert(listObjects.IsTruncated, Equals, false)
	c.Assert(listObjects.CommonPrefixes[0], Equals, "obj1")

	// test list objects with only delimiter
	listObjects, err = donut.ListObjects("foo", "", "", "1", 10)
	c.Assert(err, IsNil)
	_, ok := listObjects.Objects["obj2"]
	c.Assert(ok, Equals, true)
	c.Assert(listObjects.IsTruncated, Equals, false)
	c.Assert(listObjects.CommonPrefixes[0], Equals, "obj1")

	// test list objects with only prefix
	listObjects, err = donut.ListObjects("foo", "o", "", "", 10)
	c.Assert(err, IsNil)
	c.Assert(listObjects.IsTruncated, Equals, false)
	_, ok1 := listObjects.Objects["obj1"]
	_, ok2 := listObjects.Objects["obj2"]
	c.Assert(ok1, Equals, true)
	c.Assert(ok2, Equals, true)

	three := ioutil.NopCloser(bytes.NewReader([]byte("three")))
	metadata["contentLength"] = strconv.Itoa(len("three"))
	_, err = donut.PutObject("foo", "obj3", "", three, metadata)
	c.Assert(err, IsNil)

	obj3, size, err := donut.GetObject("foo", "obj3")
	c.Assert(err, IsNil)
	c.Assert(size, Equals, int64(len([]byte("three"))))

	var readerBuffer3 bytes.Buffer
	_, err = io.CopyN(&readerBuffer3, obj3, size)
	c.Assert(err, IsNil)
	c.Assert(readerBuffer3.Bytes(), DeepEquals, []byte("three"))

	// test list objects with maxkeys
	listObjects, err = donut.ListObjects("foo", "o", "", "", 2)
	c.Assert(err, IsNil)
	c.Assert(listObjects.IsTruncated, Equals, true)
	c.Assert(len(listObjects.Objects), Equals, 2)
}
