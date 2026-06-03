# Go portable packer

这是一个用 Go 写的单文件打包器示例，用来替代 RustDesk `libs/portable` 的思路。

它分成两个程序：

- `stub`: 最终运行壳，负责从自身 exe 末尾读取 payload，释放文件并启动入口程序。
- `packer`: 打包工具，负责把目标目录压缩成 `tar.gz`，追加到 `stub.exe` 末尾，生成最终单文件 exe。

这样做的好处是：`stub.exe` 和 `packer.exe` 编译好以后，以后重新打包其他程序时不需要 Rust，也不需要 Go，只需要运行 `packer.exe`。

## 编译

在本目录执行：

```powershell
go build -o .\bin\stub.exe .\cmd\stub
go build -o .\bin\packer.exe .\cmd\packer
```

如果要给最终生成的单文件 exe 设置图标，先准备一个 `.ico` 文件，然后用脚本编译 `stub.exe`：

```powershell
.\scripts\build-windows.ps1 -Icon .\assets\app.ico
```

脚本会先用 `windres` 生成 `cmd\stub\rsrc.syso`，Go 编译 `stub.exe` 时会自动嵌入这个资源。之后用这个 `stub.exe` 打包出的最终 exe 会继承同一个图标。

如果不想运行时显示控制台窗口，可以这样编译 Windows GUI 壳：

```powershell
go build -ldflags="-H=windowsgui" -o .\bin\stub.exe .\cmd\stub
go build -o .\bin\packer.exe .\cmd\packer
```

带图标并隐藏控制台窗口时，可以这样编译：

```powershell
.\scripts\build-windows.ps1 -Icon .\assets\app.ico -Gui
```

## 打包程序

假设目标目录如下：

```text
D:\myapp\dist
  myapp.exe
  xxx.dll
  config.json
  assets\
```

执行：

```powershell
.\bin\packer.exe `
  -stub .\bin\stub.exe `
  -folder "D:\myapp\dist" `
  -entry "myapp.exe" `
  -out ".\bin\myapp-portable.exe" `
  -app "myapp"
```

生成的 `myapp-portable.exe` 就是最终单文件。

如果入口程序需要管理员权限，增加 `-admin`：

```powershell
.\bin\packer.exe `
  -stub .\bin\stub.exe `
  -folder "D:\myapp\dist" `
  -entry "myapp.exe" `
  -out ".\bin\myapp-portable.exe" `
  -app "myapp" `
  -admin
```

带 `-admin` 生成的单文件在双击时会自动弹出 Windows UAC 提权提示。

## 运行逻辑

最终 exe 运行时会：

1. 读取自身末尾的 payload footer
2. 定位 metadata 和 tar.gz payload
3. 解压到系统缓存目录下的应用目录
4. 启动 metadata 中记录的入口程序
5. 把命令行参数透传给入口程序

Windows 下释放目录一般类似：

```text
%LOCALAPPDATA%\myapp
```

## 和 RustDesk portable 的区别

- 不需要每次打包都重新编译壳程序。
- 不使用 `include_bytes!`，而是把 payload 追加到 exe 末尾。
- 不包含 RustDesk 专用的 `install.exe`、quick support、RuntimeBroker 逻辑。
- 使用 Go 标准库的 `tar.gz`，不依赖 brotli。

## 注意事项

- 这不是内存加载方案，文件仍会释放到磁盘后运行。
- 当前示例每次启动都会清理并重新释放目录。
- 如果目标程序要求写入自身目录，释放目录必须有写权限。
- 如果要正式发布，建议给最终 exe 做代码签名。
