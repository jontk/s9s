PLUGIN_NAME = hello
PLUGIN_FILE = $(PLUGIN_NAME).so

.PHONY: all clean

all: $(PLUGIN_FILE)

$(PLUGIN_FILE): main.go
	go build -buildmode=plugin -o $(PLUGIN_FILE) main.go

clean:
	rm -f $(PLUGIN_FILE)

install: $(PLUGIN_FILE)
	mkdir -p ../../../../plugins
	cp $(PLUGIN_FILE) ../../../../plugins/