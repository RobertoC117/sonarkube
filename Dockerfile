FROM golang:1.27 AS build
WORKDIR /src
#copy go.mod and go.sum for leverage docker cache
COPY go.mod go.sum ./
# download dependencies
RUN go mod download
COPY . .
# The binary is created in /src/bin/sonarkube, copy it to /bin for export
RUN go build -o /bin/sonarkube .

FROM scratch
#Copy the binary from the build stage
COPY --from=build /bin/sonarkube /bin/sonarkube
# execution
CMD ["/bin/sonarkube"]