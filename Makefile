BUILD_DIR := ./build

clean:
	rm -rf $(BUILD_DIR)

build:clean
	go build -v -tags production -o $(BUILD_DIR)/viewer ./cmd/viewer/.
	go build -v -tags production -o $(BUILD_DIR)/loggen ./cmd/loggen/.