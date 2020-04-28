# otecstar

Simple network status monitor for OTECStar devices.

# What

Some ISPs install OTECStar devices for users to access their internet service. However, either their service sucks, or these devices suck, the Internet connection is not stable (sometimes not usable at all), so I need a simple way to know whether I'm currently connected or not.

# How

## To build

```shell script
go build
```

## To config

```shell script
cp config_sample.ini config.ini
```

And edit `config.ini`, replace router IP, username and password.

## To run

```shell script
./otecstar
```

## How does it look like?

![Screenshot](./screenshot.png)

# License

GPL-3.0.