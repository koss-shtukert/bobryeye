# --- Stage 1: Build OpenCV from source (ARM64 compatible) ---
FROM arm64v8/ubuntu:22.04 AS opencv-builder

ARG OPENCV_VERSION=4.8.1
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    build-essential cmake git wget unzip pkg-config \
    libjpeg-dev libpng-dev libtiff-dev \
    libavcodec-dev libavformat-dev libswscale-dev \
    libv4l-dev libxvidcore-dev libx264-dev \
    libgtk-3-dev libatlas-base-dev gfortran \
    python3-dev ffmpeg && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /opt
RUN git clone --branch ${OPENCV_VERSION} https://github.com/opencv/opencv.git && \
    git clone --branch ${OPENCV_VERSION} https://github.com/opencv/opencv_contrib.git

WORKDIR /opt/opencv/build
RUN cmake -D CMAKE_BUILD_TYPE=Release \
          -D CMAKE_INSTALL_PREFIX=/usr/local \
          -D OPENCV_EXTRA_MODULES_PATH=/opt/opencv_contrib/modules \
          -D BUILD_EXAMPLES=OFF \
          -D BUILD_TESTS=OFF \
          -D BUILD_PERF_TESTS=OFF \
          -D BUILD_opencv_python=OFF \
          -D BUILD_DOCS=OFF \
          -D WITH_TBB=ON \
          -D WITH_V4L=ON \
          -D WITH_FFMPEG=ON .. && \
    make -j$(nproc) && \
    make install && \
    ldconfig

# --- Stage 2: Build Go binary with gocv ---
FROM golang:1.22-bookworm AS go-builder

# Copy OpenCV from builder stage
COPY --from=opencv-builder /usr/local /usr/local

# Set env for pkg-config
ENV PKG_CONFIG_PATH=/usr/local/lib/pkgconfig
RUN ldconfig

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -tags customenv -o bobryeye .

# --- Stage 3: Final image (runtime only) ---
FROM arm64v8/debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    libgtk-3-0 libavcodec59 libavformat59 libswscale6 \
    libv4l-0 libxvidcore4 libx264-dev libjpeg62-turbo \
    libpng16-16 libtiff6 libopenexr-dev libtbb12 \
    libgstreamer1.0-0 libgstreamer-plugins-base1.0-0 \
    ffmpeg && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=go-builder /app/bobryeye /app/bobryeye
COPY config.yaml /app/config.yaml

CMD ["./bobryeye"]
