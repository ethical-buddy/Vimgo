#!/bin/bash

# Build the file manager
go build main.go

# Move the executable to /usr/local/bin
sudo mv main /usr/local/bin/vimgo

# Make the executable runnable by everyone
sudo chmod +x /usr/local/bin/vimgo

# Add an alias to run the file manager
echo "alias fm='vimgo'" >> ~/.bashrc
source ~/.bashrc

echo "vimgo installed successfully!"
