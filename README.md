# GVM - Go Version Manager

一个简单易用的Go版本管理工具，基于Cobra框架开发。

## 功能特性

- ✅ **多版本管理**: 同时安装和管理多个Go版本
- ✅ **快速切换**: 在不同Go版本间轻松切换
- ✅ **版本列表**: 查看已安装和可安装的Go版本
- ✅ **简洁界面**: 用户友好的命令行界面
- ✅ **跨平台**: 支持Windows、macOS和Linux

## 安装

### 从源码安装

```bash
# 克隆仓库
git clone https://github.com/philokun/gvm.git
cd gvm

# 构建
go build -o gvm main.go

# 安装到系统路径
sudo mv gvm /usr/local/bin/
```

### 快速安装

```bash
curl -sSL https://raw.githubusercontent.com/philokun/gvm/main/install.sh | bash
```

## 使用方法

### 查看帮助
```bash
gvm --help
```

### 列出已安装的版本
```bash
gvm list
```

### 查看可用的Go版本
```bash
gvm available
```

### 安装特定版本
```bash
# 安装Go 1.21.5
gvm install go1.21.5

# 或者使用简写形式
gvm install 1.21.5
```

### 切换到特定版本
```bash
# 切换到Go 1.21.5
gvm use go1.21.5

# 或者使用简写形式
gvm use 1.21.5
```

### 查看当前版本
```bash
gvm current
```

### 卸载版本
```bash
gvm uninstall go1.21.5
```

## 命令列表

| 命令 | 描述 |
|------|------|
| `gvm list` | 列出已安装的Go版本 |
| `gvm available` | 列出可安装的Go版本 |
| `gvm install <version>` | 安装指定版本的Go |
| `gvm use <version>` | 切换到指定版本的Go |
| `gvm current` | 显示当前使用的Go版本 |
| `gvm uninstall <version>` | 卸载指定版本的Go |
| `gvm --help` | 显示帮助信息 |

## 技术架构

### 项目结构
```
gvm/
├── cmd/                    # Cobra命令定义
│   ├── root.go            # 根命令
│   ├── list.go            # 列出版本命令
│   ├── install.go         # 安装版本命令
│   ├── use.go             # 切换版本命令
│   ├── uninstall.go       # 卸载版本命令
│   ├── current.go         # 显示当前版本命令
│   └── available.go       # 显示可用版本命令
├── internal/              # 内部模块
│   ├── version/           # 版本管理核心
│   │   └── version.go     # 版本管理实现
│   ├── config/            # 配置管理
│   │   └── config.go      # 配置文件处理
│   ├── utils/             # 工具函数
│   │   └── utils.go       # 下载、解压等工具
│   └── output/            # 输出格式化
│       └── output.go      # 友好的输出格式
├── main.go                # 程序入口
├── go.mod                   # Go模块定义
└── README.md              # 项目文档
```

### 核心功能模块

#### 1. 版本管理模块 (`internal/version/`)
- 获取可用版本列表
- 下载和安装Go版本
- 版本切换管理
- 已安装版本查询

#### 2. 配置管理模块 (`internal/config/`)
- 配置文件读写
- 当前版本记录
- 安装目录管理

#### 3. 工具模块 (`internal/utils/`)
- 文件下载
- 压缩包解压
- 系统信息获取
- Shell配置更新

#### 4. 输出模块 (`internal/output/`)
- 彩色输出
- 错误处理
- 用户交互

## 开发计划

- [ ] **版本1.1**: 添加版本缓存功能
- [ ] **版本1.2**: 支持代理设置
- [ ] **版本1.3**: 添加版本别名功能
- [ ] **版本1.4**: 支持项目级别的版本管理
- [ ] **版本1.5**: 添加Web界面管理

## 贡献

欢迎提交Issue和Pull Request！

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 作者

PhiloKun - philokun@example.com