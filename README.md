# 编写 Web 应用程序

> 原文：[Writing Web Applications - The Go Programming Language](https://go.dev/doc/articles/wiki/)

---

## 前言

在浏览go的官方文档的时候看到了这篇web教学，我通篇阅读并且动手实现后，发现这篇文档是不错的学习资料。但是没有汉化的情况下，对于英语阅读能力一般的读者来说是个不小的挑战。所以我翻译并且加入了一些自己的修改，对一些知识进行了补充说明。修改后文档更为简洁易懂，并且保留了精华部分。

需要完整代码实现的，请移步 [GoWiki实现]([Errrori/GoWiki](https://github.com/Errrori/GoWiki))

---

## 简介和前置知识

本文涵盖

- 使用 load 和 save 方法创建数据结构
- 使用 `net/http` 包构建 Web 应用程序
- 使用 `html/template` 包处理 HTML 模板
- 使用 `regexp` 包验证用户输入
- 使用闭包

最终成果

- 一个简单的wiki页面，包括添加，修改，保存文档的功能

前置知识（不懂也没关系，大多数比较简单，可以自行搜索学习）

- 结构体、方法、指针等go的基本语法概念

- 基本的web和html知识

- `fmt`,`os`,`errors`包的简单使用

---

## 一些提醒

程序构建时，在命令行执行：

```bash
go build wiki.go
```

来创建可执行文件。但是接下来的执行命令随执行环境而异：

```bash
# Windows环境
wiki
# Linux/Unix环境
./wiki
```

---

## 开始

创建一个名为 `wiki.go` 的文件，添加以下代码：

```go
package main

import (
    "fmt"
    "os"
)
```

我们从 Go 标准库中导入了 `fmt` 和 `os` 包。稍后，当我们实现其他功能时，会向 `import` 声明中添加更多包。

---

## 数据结构 Page

一个 wiki 由一系列相互连接的页面组成，每个页面都有一个标题（title）和一个正文（body，即页面内容）。在这里，我们将 `Page` 定义为一个结构体，其中包含两个字段，分别表示标题和正文。

```go
type Page struct {
    Title string
    Body  []byte
}
```

类型 `[]byte` 表示"一个 `byte` 切片"（参见 [Slices: usage and internals](https://go.dev/doc/articles/slices_usage_and_internals.html) 了解更多关于切片的内容）。`Body` 元素是 `[]byte` 而不是 `string`，因为这是我们将使用的 `io` 库所期望的类型，正如下面你将看到的。

`Page` 结构体描述了页面数据如何在内存中存储。但持久化存储呢？我们可以通过在 `Page` 上创建一个 `save` 方法来解决：

```go
func (p *Page) save() error {
    filename := p.Title + ".txt"
    return os.WriteFile(filename, p.Body, 0600)
}
```

该方法将 `Page` 的 `Body` 保存到一个文本文件中。为简单起见，我们将使用 `Title` 作为文件名。

`save` 方法返回一个 `error` 值，因为这是 `WriteFile`（一个将字节切片写入文件的标准库函数）的返回类型。`save` 方法返回错误值，以便让应用程序在写文件出现问题时处理它。如果一切顺利，`Page.save()` 将返回 `nil`（指针、接口和其他一些类型的零值）。

八进制整数字面量 `0600`（作为第三个参数传递给 `WriteFile`）表示文件应该以仅当前用户可读写的方式创建。

下面是一些常见的文件权限位：

| 八进制值   | 权限含义             | 典型使用场景            |
| ------ | ---------------- | ----------------- |
| `0644` | 拥有者读写，组和其他只读     | 普通文件：配置文件、网页、图片等  |
| `0755` | 拥有者读写执行，组和其他只读执行 | 可执行程序、脚本、目录       |
| `0666` | 所有人可读写           | 临时文件（会受 umask 影响） |
| `0700` | 仅拥有者读写执行         | 私密脚本、需要执行的程序      |
| `0400` | 仅拥有者只读           | 只读数据文件（很少用于新创建）   |

除了保存页面，我们还需要加载页面：

```go
func loadPage(title string) *Page {
    filename := title + ".txt"
    body, _ := os.ReadFile(filename)
    return &Page{Title: title, Body: body}
}
```

函数 `loadPage` 从 title 参数构造文件名，将文件内容读入新变量 `body`，并返回一个指向 `Page` 字面量的指针，该字面量使用正确的 title 和 body 值构造。

函数可以返回多个值。标准库函数 `os.ReadFile` 返回 `[]byte` 和 `error`。在 `loadPage` 中，目前还没有处理错误；下划线（`_`）符号表示的"空白标识符"用于丢弃错误返回值（本质上，将值赋给空）。

但是，如果 `ReadFile` 遇到错误会发生什么？例如，文件可能不存在。我们不应该忽略这样的错误。让我们修改函数以同时返回 `*Page` 和 `error`：

```go
func loadPage(title string) (*Page, error) {
    filename := title + ".txt"
    body, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    return &Page{Title: title, Body: body}, nil
}
```

此函数的调用者现在可以检查第二个参数；如果它是 `nil`，则表示页面加载成功。如果不是，它将是一个 `error`，可以由调用者处理（详见[语言规范](https://go.dev/ref/spec#Errors)）。

此时，我们有了一个简单的数据结构和保存/加载文件的能力。让我们编写一个 `main` 函数来测试我们所写的内容：

```go
func main() {
    p1 := &Page{Title: "TestPage", Body: []byte("This is a sample Page.")}
    p1.save()
    p2, _ := loadPage("TestPage")
    fmt.Println(string(p2.Body))
}
```

编译并执行此代码后，将创建一个名为 `TestPage.txt` 的文件，其中包含 `p1` 的内容。然后该文件将被读入结构体 `p2`，并将其 `Body` 元素打印到屏幕上。

你可以这样编译和运行程序：

```
$ go build wiki.go
$ ./wiki
This is a sample Page.
```

（如果你使用 Windows，运行程序时必须输入 `wiki`，不加 `./`。）

---

## 介绍 `net/http` 包（插曲）

下面是一个简单的 Web 服务器完整工作示例：

```go
//go:build ignore

package main

import (
    "fmt"
    "log"
    "net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func main() {
    http.HandleFunc("/", handler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

`main` 函数以调用 `http.HandleFunc` 开始，它告诉 `http` 包使用 `handler` 处理所有对 Web 根路径（`"/"`）的请求。

然后它调用 `http.ListenAndServe`，指定它应该在所有网络接口上监听 8080 端口（`":8080"`）。（目前不必担心第二个参数 `nil`。）这个函数将阻塞直到程序终止。

`ListenAndServe` 总是返回一个错误，因为它只在发生意外错误时才返回。为了记录该错误，我们用 `log.Fatal` 包装函数调用。

函数 `handler` 的类型是 `http.HandlerFunc`。它以 `http.ResponseWriter` 和 `http.Request` 作为参数。

`http.ResponseWriter` 值用于组装 HTTP 服务器的响应；通过向其写入，我们将数据发送给 HTTP 客户端。

`http.Request` 是一个表示客户端 HTTP 请求的数据结构。`r.URL.Path` 是请求 URL 的路径部分。后面的 `[1:]` 表示"创建一个从第 1 个字符到末尾的 `Path` 子切片"。这会去掉路径名前面的 "/"。

如果运行此程序并访问 URL：

```
http://localhost:8080/monkeys
```

程序将显示一个包含以下内容的页面：

```
Hi there, I love monkeys!
```

---

## 使用 `net/http` 提供 wiki 页面服务

要使用 `net/http` 包，必须导入它：

```go
import (
    "fmt"
    "os"
    "log"
    "net/http"
)
```

让我们创建一个 handler——`viewHandler`，允许用户查看 wiki 页面。它将处理以 "/view/" 为前缀的 URL。

```go
func viewHandler(w http.ResponseWriter, r *http.Request) {
    title := r.URL.Path[len("/view/"):]
    p, _ := loadPage(title)
    fmt.Fprintf(w, "<h1>%s</h1><div>%s</div>", p.Title, p.Body)
}
```

再次注意使用 `_` 忽略 `loadPage` 的 `error` 返回值。这里为简单起见这样做，通常被认为是不好的做法，稍后会处理。

首先，该函数从 `r.URL.Path`（请求 URL 的路径部分）中提取页面标题。使用 `[len("/view/"):]` 对 `Path` 进行重新切片，以去掉请求路径前面的 `"/view/"` 部分。这是因为路径总是以 `"/view/"` 开头，而这不是页面标题的一部分。

然后，该函数加载页面数据，用简单的 HTML 字符串格式化页面，并将其写入 `w`（即 `http.ResponseWriter`）。

要使用此 handler，我们重写 `main` 函数以初始化 `http`，使用 `viewHandler` 处理 `/view/` 路径下的任何请求：

```go
func main() {
    http.HandleFunc("/view/", viewHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

让我们创建一些页面数据（如 `test.txt`），编译代码，然后尝试提供 wiki 页面服务。

在编辑器中打开 `test.txt` 文件，将字符串 "Hello world"（不含引号）保存在其中。

```
$ go build wiki.go
$ ./wiki
```

运行此 Web 服务器后，访问 `http://localhost:8080/view/test` 应该显示一个标题为"test"、内容为"Hello world"的页面。

---

## 编辑页面

没有编辑页面的能力，wiki 就不是 wiki。让我们创建两个新的 handler：一个名为 `editHandler` 用于显示"编辑页面"表单，另一个名为 `saveHandler` 用于保存通过表单输入的数据。

首先，将它们添加到 `main()` 中：

```go
func main() {
    http.HandleFunc("/view/", viewHandler)
    http.HandleFunc("/edit/", editHandler)
    http.HandleFunc("/save/", saveHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

`editHandler` 函数加载页面（或者，如果页面不存在，创建一个空的 `Page` 结构体），并显示一个 HTML 表单：

```go
func editHandler(w http.ResponseWriter, r *http.Request) {
    title := r.URL.Path[len("/edit/"):]
    p, err := loadPage(title)
    if err != nil {
        p = &Page{Title: title}
    }
    fmt.Fprintf(w, "<h1>Editing %s</h1>"+
        "<form action=\"/save/%s\" method=\"POST\">"+
        "<textarea name=\"body\">%s</textarea><br>"+
        "<input type=\"submit\" value=\"Save\">"+
        "</form>",
        p.Title, p.Title, p.Body)
}
```

这个函数可以正常工作，但所有这些硬编码的 HTML 都很丑陋。当然，有更好的方法。

---

## `html/template` 包

`html/template` 包是 Go 标准库的一部分。我们可以使用 `html/template` 将 HTML 保存在单独的文件中，这样就能在不修改底层 Go 代码的情况下更改编辑页面的布局。

首先，必须将 `html/template` 添加到导入列表中。我们也不再用到 `fmt`，所以要将其删除：

```go
import (
    "html/template"
    "os"
    "net/http"
)
```

让我们创建一个包含 HTML 表单的模板文件。打开一个新文件命名为 `edit.html`，添加以下内容：

```html
<h1>Editing {{.Title}}</h1>

<form action="/save/{{.Title}}" method="POST">
<div><textarea name="body" rows="20" cols="80">{{printf "%s" .Body}}</textarea></div>
<div><input type="submit" value="Save"></div>
</form>
```

修改 `editHandler` 以使用模板，而不是硬编码的 HTML：

```go
func editHandler(w http.ResponseWriter, r *http.Request) {
    title := r.URL.Path[len("/edit/"):]
    p, err := loadPage(title)
    if err != nil {
        p = &Page{Title: title}
    }
    t, _ := template.ParseFiles("edit.html")
    t.Execute(w, p)
}
```

`template.ParseFiles` 函数将读取 `edit.html` 的内容并返回一个 `*template.Template`。

`t.Execute` 方法执行模板，将生成的 HTML 写入 `http.ResponseWriter`。`.Title` 和 `.Body` 点标识符引用的是 `p.Title` 和 `p.Body`。

模板指令用双大括号包裹。`printf "%s" .Body` 指令是一个函数调用，它将 `.Body` 作为字符串而不是字节流输出，与调用 `fmt.Printf` 的效果相同。`html/template` 包有助于确保模板动作生成的 HTML 是安全且外观正确的。例如，它会自动转义大于号（`>`），将其替换为 `&gt;`，以确保用户数据不会破坏表单 HTML。

由于我们正在使用模板，现在为 `viewHandler` 也创建一个模板，命名为 `view.html`：

```html
<h1>{{.Title}}</h1>

<p>[<a href="/edit/{{.Title}}">edit</a>]</p>

<div>{{printf "%s" .Body}}</div>
```

相应地修改 `viewHandler`：

```go
func viewHandler(w http.ResponseWriter, r *http.Request) {
    title := r.URL.Path[len("/view/"):]
    p, _ := loadPage(title)
    t, _ := template.ParseFiles("view.html")
    t.Execute(w, p)
}
```

注意到我们在两个 handler 中使用了几乎完全相同的模板代码。让我们通过将模板代码移到自己的函数中来消除这种重复：

```go
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
    t, _ := template.ParseFiles(tmpl + ".html")
    t.Execute(w, p)
}
```

并修改 handler 以使用该函数：

```go
func viewHandler(w http.ResponseWriter, r *http.Request) {
    title := r.URL.Path[len("/view/"):]
    p, _ := loadPage(title)
    renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
    title := r.URL.Path[len("/edit/"):]
    p, err := loadPage(title)
    if err != nil {
        p = &Page{Title: title}
    }
    renderTemplate(w, "edit", p)
}
```

---

## 处理不存在的页面

如果访问 `/view/APageThatDoesntExist` 会怎样？你将看到一个包含 HTML 的页面。这是因为它忽略了 `loadPage` 返回的错误值，并继续尝试在没有数据的情况下填充模板。相反，如果请求的页面不存在，应该将客户端重定向到编辑页面，以便可以创建内容：

```go
func viewHandler(w http.ResponseWriter, r *http.Request) {
    title := r.URL.Path[len("/view/"):]
    p, err := loadPage(title)
    if err != nil {
        http.Redirect(w, r, "/edit/"+title, http.StatusFound)
        return
    }
    renderTemplate(w, "view", p)
}
```

`http.Redirect` 函数向 HTTP 响应添加一个 HTTP 状态码 `http.StatusFound`（302）和一个 `Location` 头。

---

## 保存页面

`saveHandler` 函数将处理编辑页面上表单的提交。取消 `main` 中相关行的注释后，让我们实现该 handler：

```go
func saveHandler(w http.ResponseWriter, r *http.Request) {
    title := r.URL.Path[len("/save/"):]
    body := r.FormValue("body")
    p := &Page{Title: title, Body: []byte(body)}
    p.save()
    http.Redirect(w, r, "/view/"+title, http.StatusFound)
}
```

页面标题（从 URL 中获取）和表单的唯一字段 `Body` 被存储在一个新的 `Page` 中。然后调用 `save()` 方法将数据写入文件，并将客户端重定向到 `/view/` 页面。

`FormValue` 返回的值是 `string` 类型。我们必须将该值转换为 `[]byte` 才能匹配 `Page` 结构体。我们使用 `[]byte(body)` 来执行转换。

---

## 错误处理

程序中多处忽略了错误。这是不好的做法，尤其是因为当错误确实发生时，程序将产生意外行为。更好的解决方案是处理错误并向用户返回错误消息。这样，如果出现任何问题，服务器将按照我们希望的方式运行，并且用户可以收到通知。

首先，让我们处理 `renderTemplate` 中的错误：

```go
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
    t, err := template.ParseFiles(tmpl + ".html")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    err = t.Execute(w, p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
```

`http.Error` 函数发送指定的 HTTP 响应码（此处为"Internal Server Error"）和错误消息。现在将这部分代码放在单独函数中的决策已经得到回报。

现在来修复 `saveHandler`：

```go
func saveHandler(w http.ResponseWriter, r *http.Request) {
    title := r.URL.Path[len("/save/"):]
    body := r.FormValue("body")
    p := &Page{Title: title, Body: []byte(body)}
    err := p.save()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    http.Redirect(w, r, "/view/"+title, http.StatusFound)
}
```

`p.save()` 期间发生的任何错误都将报告给用户。

---

## 模板缓存

代码中存在一个低效之处：`renderTemplate` 每次渲染页面时都会调用 `ParseFiles`。更好的方法是在程序初始化时调用一次 `ParseFiles`，将所有模板解析到一个 `*Template` 中。然后我们可以使用 `ExecuteTemplate` 方法来渲染特定的模板。

首先创建一个名为 `templates` 的全局变量，并用 `ParseFiles` 初始化它：

```go
var templates = template.Must(template.ParseFiles("edit.html", "view.html"))
```

`template.Must` 函数是一个便捷包装器，当传入非 nil 的 `error` 值时它会触发 panic，否则原样返回 `*Template`。panic 在这里是合适的；如果模板无法加载，唯一合理的做法就是退出程序。

`ParseFiles` 函数接收任意数量的字符串参数来标识我们的模板文件，并将这些文件解析为以文件名命名的模板。如果我们要向程序添加更多模板，只需将它们的名称添加到 `ParseFiles` 调用的参数中即可。

然后修改 `renderTemplate` 函数，使用适当的模板名称调用 `templates.ExecuteTemplate` 方法：

```go
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
    err := templates.ExecuteTemplate(w, tmpl+".html", p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
```

请注意，模板名称就是模板文件名，所以必须在 `tmpl` 参数后面加上 `".html"`。

---

## 验证

你可能已经注意到，这个程序有一个严重的安全缺陷：用户可以提供任意路径在服务器上读写。为了缓解这个问题，我们可以编写一个函数，用正则表达式来验证标题。

首先，将 `"regexp"` 添加到 `import` 列表中。然后创建一个全局变量来存储我们的验证表达式：

```go
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
```

`regexp.MustCompile` 函数将解析和编译正则表达式，并返回一个 `regexp.Regexp`。`MustCompile` 与 `Compile` 的区别在于，如果表达式编译失败，它会触发 panic，而 `Compile` 返回一个 `error` 作为第二个参数。

现在，让我们编写一个使用 `validPath` 表达式来验证路径并提取页面标题的函数：

```go
func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
    m := validPath.FindStringSubmatch(r.URL.Path)
    if m == nil {
        http.NotFound(w, r)
        return "", errors.New("invalid Page Title")
    }
    return m[2], nil // The title is the second subexpression.
}
```

> #### 对正则表达式 validPath 的解释：

- **`^`**：匹配字符串开头。

- **`/`**：匹配字面斜杠。

- **`(edit|save|view)`**：**捕获组 1**，匹配其中任意一个单词：`edit`、`save` 或 `view`。

- **`/`**：再匹配一个斜杠。

- **`([a-zA-Z0-9]+)`**：**捕获组 2**，匹配一个或多个大小写字母或数字，这就是合法的页面标题。

- **`$`**：匹配字符串结尾。

> #### 为什么获取标题使用 m[2] ?

`validPath.FindStringSubmatch(r.URL.Path)` 返回一个 `[]string` 切片：

- `m[0]`：整个匹配到的字符串（如 `/view/TestPage`）

- `m[1]`：第一个捕获组的内容（`edit` / `save` / `view`）

- `m[2]`：第二个捕获组的内容，即**页面标题**（`TestPage`）

如果标题有效，它将与一个 `nil` 错误值一起返回。如果标题无效，该函数将向 HTTP 连接写入"404 Not Found"错误，并向 handler 返回一个错误。要创建一个新的错误，我们必须导入 `errors` 包。

让我们在每个 handler 中添加对 `getTitle` 的调用：

```go
func viewHandler(w http.ResponseWriter, r *http.Request) {
    title, err := getTitle(w, r)
    if err != nil {
        return
    }
    p, err := loadPage(title)
    if err != nil {
        http.Redirect(w, r, "/edit/"+title, http.StatusFound)
        return
    }
    renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
    title, err := getTitle(w, r)
    if err != nil {
        return
    }
    p, err := loadPage(title)
    if err != nil {
        p = &Page{Title: title}
    }
    renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
    title, err := getTitle(w, r)
    if err != nil {
        return
    }
    body := r.FormValue("body")
    p := &Page{Title: title, Body: []byte(body)}
    err = p.save()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    http.Redirect(w, r, "/view/"+title, http.StatusFound)
}
```

---

## 介绍函数字面量和闭包

在每个 handler 中捕获错误条件会引入大量重复代码。如果我们能将每个 handler 包装在一个执行此验证和错误检查的函数中呢？Go 的[函数字面量](https://go.dev/ref/spec#Function_literals)提供了一种强大的抽象功能方法，可以帮助我们。

首先，我们重写每个 handler 的函数定义，使其接受一个 title 字符串：

```go
func viewHandler(w http.ResponseWriter, r *http.Request, title string)
func editHandler(w http.ResponseWriter, r *http.Request, title string)
func saveHandler(w http.ResponseWriter, r *http.Request, title string)
```

现在定义一个包装函数，它**接收一个上述类型的函数**，并返回一个 `http.HandlerFunc` 类型的函数（适合传递给 `http.HandleFunc` 函数）：

```go
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 这里我们将从 Request 中提取页面标题，
        // 并调用提供的 handler 'fn'
    }
}
```

返回的函数被称为**闭包**，因为它包含了在其外部定义的值。这种情况下，变量 `fn`（即 `makeHandler` 的唯一参数）被闭包封闭。变量 `fn` 将是我们的 save、edit 或 view handler 之一。

现在我们可以从 `getTitle` 中取出代码并在这里使用（稍作修改）：

```go
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        m := validPath.FindStringSubmatch(r.URL.Path)
        if m == nil {
            http.NotFound(w, r)
            return
        }
        fn(w, r, m[2])
    }
}
```

`makeHandler` 返回的闭包是一个接收 `http.ResponseWriter` 和 `http.Request` 的函数（换句话说，是一个 `http.HandlerFunc`）。该闭包从请求路径中提取 `title`，并使用 `validPath` 正则表达式验证它。如果 `title` 无效，将使用 `http.NotFound` 函数向 `ResponseWriter` 写入错误。如果 `title` 有效，则使用 `ResponseWriter`、`Request` 和 `title` 作为参数调用被封闭的 handler 函数 `fn`。

现在我们可以用 `makeHandler` 在 `main` 中包装 handler 函数，然后将它们注册到 `http` 包：

```go
func main() {
    http.HandleFunc("/view/", makeHandler(viewHandler))
    http.HandleFunc("/edit/", makeHandler(editHandler))
    http.HandleFunc("/save/", makeHandler(saveHandler))

    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

最后，我们从 handler 函数中移除对 `getTitle` 的调用，使它们变得更简洁：

```go
func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
    p, err := loadPage(title)
    if err != nil {
        http.Redirect(w, r, "/edit/"+title, http.StatusFound)
        return
    }
    renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
    p, err := loadPage(title)
    if err != nil {
        p = &Page{Title: title}
    }
    renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
    body := r.FormValue("body")
    p := &Page{Title: title, Body: []byte(body)}
    err := p.save()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    http.Redirect(w, r, "/view/"+title, http.StatusFound)
}
```

---

## 试试看！

重新编译代码并运行应用程序：

```
$ go build wiki.go
$ ./wiki
```

访问 `http://localhost:8080/view/ANewPage` 应该会显示页面编辑表单。然后你应该能够输入一些文本，点击"Save"，并被重定向到新创建的页面。

---

## 其他任务

以下是一些你可能想要自己完成的简单任务：

- 将模板存储在 `tmpl/` 目录中，将页面数据存储在 `data/` 目录中
- 添加一个 handler，使 Web 根路径重定向到 `/view/FrontPage`
- 美化页面模板，使其成为有效的 HTML 并添加一些 CSS 规则
- 通过将 `[PageName]` 实例转换为 `<a href="/view/PageName">PageName</a>` 来实现页面之间的链接（提示：可以使用 `regexp.ReplaceAllFunc` 来实现）

---

*原文来源：[Writing Web Applications - The Go Programming Language](https://go.dev/doc/articles/wiki/)*
