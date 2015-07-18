## honky/audiobook-feeds

This script generates rss and atom feeds from your local file system to server e.g. audio books or other sequential audio files and servers them as a static webserver. Furthermore it offers a small gui to select single folders per podcast. 

### But why?

By that you can tranfer your audiobooks to your smart phones / tables / internet radios by podcatching functionality. For me this saves a lot of annoying cable handling time and improved my book per month rate.

### other facts

- it offers a simple password protection to prevent copy right issues
- runs in docker container (see MakeFile and DockerFile)
- works when placed in subdirs example.com/audio/ (e.g. reg for ssl without wildcard)
- allows custom theming and uses material design as default theme
- handles encoding of filenames at best effort
- further information can be found on my 

### Screenshot

![Screenshot](https://github.com/PaulPetring/audiobook-feeds/blob/master/themes/default/default.png?raw=true")


- for more information see on my [website](https://defendtheplanet.net/2015/07/18/paulpetringaudiobook-feeds/)
 
### Usage

1. adjust settings in ```config.default.json``` and save it as ```config.json``` 
    - simply skip this step to use the defaults
2. (sym)link some audiobook files as your ./files dir
    - e.g. ```ln -s /path/to/your/audiobook/collection/ ./files/```
    - or copy some of them to the ./files/ dir 
3. run the script inside a docker container using the Dockerfile provided
    - by using default docker ```sudo make build && sudo make run```
    - by adjusting and using docker without MakeFile
        -  ```docker build -t audiobook-feeds .```
        -  ```docker run --rm -v `pwd`:/usr/src/myapp -w /usr/src/myapp -it --rm -p 8080:8080 --name audiobook-feeds audiobook-feeds go run feed.go```
    - by using go on the host machine 
        - ```sudo make run-no-docker```
4. navigate to 127.0.0.1:8080/folders/ to get to the web interface
5. enjoy your audiobooks as podcasts

### Ideas to improve
    - feed image, if provided in folder
    - fix handling of the char '#'
    - different sorting algorithms

### Thanks to 

-  [Compufreak345](github.com/Compufreak345/) for guiding me through my first very own golang projects

