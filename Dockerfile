# build frontend
# 与 CI 工作流保持一致：工作流用 Node 22.10.0 + npm 验证过构建，这里复用同一个版本
ARG NODE_VERSION=22
FROM node:${NODE_VERSION}-alpine AS web_image

# 使用淘宝npm镜像源加速依赖安装
RUN npm config set registry https://registry.npmmirror.com

WORKDIR /build

# 先复制整个项目（排除.dockerignore中的文件）
COPY . /build

# 安装依赖
# 这里用 npm ci（依据 package-lock.json）而不是 pnpm，原因：
#   1) 仓库里的 pnpm-lock.yaml 是 lockfileVersion 6.0，且缺少 @vicons/ionicons5，
#      与 package.json 已不同步，pnpm 无法 --frozen-lockfile；
#   2) `npm install pnpm -g` 装到的是 pnpm 10（只支持 lockfileVersion 9.x），
#      会忽略旧锁文件重新解析依赖，再因默认拦截依赖的构建脚本报
#      ERR_PNPM_IGNORED_BUILDS，导致 `pnpm install` 直接失败。
# package-lock.json 覆盖了全部 40 个依赖且已被 CI 的 `npm ci` 验证通过。
RUN npm ci --no-audit --no-fund

# 构建项目
RUN npm run build

# build backend
# 最新alpine3.19导致sqlite3编译失败(https://github.com/mattn/go-sqlite3/issues/1164，
# 临时解决方案:https://github.com/mattn/go-sqlite3/pull/1177)
# sun-panel暂时解决方案使用golang:1.21-alpine3.18（因旧版本使用没问题，短期内较稳定） 
FROM golang:1.21-alpine3.18 AS server_image

WORKDIR /build

COPY ./service .

# 中国国内源
# RUN sed -i "s@dl-cdn.alpinelinux.org@mirrors.aliyun.com@g" /etc/apk/repositories \
#     && go env -w GOPROXY=https://goproxy.cn,direct

RUN apk add --no-cache bash curl gcc git musl-dev

RUN go env -w GO111MODULE=on \
    && export PATH=$PATH:/go/bin \
    && go install -a -v github.com/go-bindata/go-bindata/...@latest \
    && go install -a -v github.com/elazarl/go-bindata-assetfs/...@latest \
    && go-bindata-assetfs -o=assets/bindata.go -pkg=assets assets/... \
    && go build -o sun-panel --ldflags="-X sun-panel/global.RUNCODE=release -X sun-panel/global.ISDOCKER=docker" main.go



# run_image
FROM alpine

WORKDIR /app

COPY --from=web_image /build/dist /app/web

COPY --from=server_image /build/sun-panel /app/sun-panel

# 中国国内源
# RUN sed -i "s@dl-cdn.alpinelinux.org@mirrors.aliyun.com@g" /etc/apk/repositories

EXPOSE 3002

RUN apk add --no-cache bash ca-certificates su-exec tzdata \
    && chmod +x ./sun-panel \
    && ./sun-panel -config

CMD ["./sun-panel"]
