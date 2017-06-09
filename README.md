## Simple entertainment news spider

##Usage

git clone https://github.com/tanyfx/ent

cd $GOPATH/src/github.com/tanyfx/ent
dep ensure -v -update


##test news
make sure that you have the permission to read/write **/home/img/news_img**
simplely
```
cd /home
sudo mkdir -p /home/img/news_img
sudo chmod 777 -R /home/img
```

```
cd $GOPATH/src/github.com/tanyfx/ent/app/newsapp/test-news
go build
./test-news
```


##test video
```
cd $GOPATH/src/github.com/tanyfx/ent/app/videoapp/test-video
go build
./test-video
```