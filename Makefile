

TARGET:=koiImport

AUTHDIR=./cmd
AUTHSRC=$(wildcard $(AUTHDIR)/*go)

$(TARGET): $(AUTHSRC)
	go build -o $(TARGET) $(AUTHDIR)

.PHONY: clean realclean
clean:
	rm -f $(TARGET)
	go clean
	find . -type f -iname \*~ -exec rm -f {} \;

realclean: clean
	go clean -modcache 


