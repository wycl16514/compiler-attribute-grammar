上一节我们研究了增强语法，本节我们看看何为属性语法。属性语法实则是在语法规则上附带上一些重要的解析信息，随着语法解析的进行，我们可以利用附带的解析信息去进行一系列操作，例如利用解析信息实现代码生成。我们先看属性语法的一个实例：
```go
NUMBER("156", 156)
```
NUMBER 是语法解析中的终结符，他附带有两个属性，一个是该标签对应字符串的内容“156”，另一个是他对应的数值也就是 156，如果符号是 ID，也就是变量，那么它可以附带一个属性就是一个指针，指向符号表的入口，该符号表包含了该变量的字符串名称，该变量对应的数据等等。

属性信息分为两种，一种是继承属性，也就是属性从语法表达式箭头左边的符号传递给右边的符号，另一种是综合属性，属性信息从箭头右边符号汇总后传递给左边符号。从前面代码中我们看到，语法解析本质上就是函数的调用，例如语法：
```go
expr -> term expr_prime
```
对应的代码实现就是：
```
expr() {
    term()
    expr_prime()
   }
```
对于继承属性，那就是父函数expr 在调用是被输入了某些参数，这些参数再传递给里面的 term,和 expr_prime，例如：
```go
expr(param) {
    term(param)
    expr_prime(param)
}
```
而综合属性就是子函数有返回值，父函数获取子函数的返回值后综合起来处理，例如：
```
expr() {
    val_term := term()
    val_expr_prime := expr_prime(param)
    do_something(val_term, val_expr_prime)
    }
```
在上一节我们使用增强语法来生成代码时，代码生成所需要的信息例如寄存器等，是从全局函数或全局变量（例如全局寄存器数组等）中获取，在属性语法中我们就可以把这些信息作为参数传递给特定的语法解析函数，这样在生成代码时就能更灵活。我们看具体的实现你就能更明白什么叫属性语法，我们还是利用上一节识别算术表达式的语法：
```go
stmt -> epsilon | expr SEMI stmt
expr -> term expr_prime
expr_prime -> PLUS term expr_prime
term -> factor term_prime
term_prime -> MUL factor term_prime | epsilon
factor -> NUMBER | LEFT_PAREN expr RIGHT_PAREN
```
在原有项目中创建新文件夹 attribute_parser,在里面创建文件 attribute_parser.go，添加代码如下：
```go
package attribute_parser

import (
	"fmt"
	"lexer"
)

type AttributeParser struct {
	parserLexer  lexer.Lexer
	reverseToken []lexer.Token
	//用于存储虚拟寄存器的名字
	registerNames []string
	//存储当前已分配寄存器的名字
	regiserStack []string
	//当前可用寄存器名字的下标
	registerNameIdx int
}

func NewAttributeParser(parserLexer lexer.Lexer) *AttributeParser {
	return &AttributeParser{
		parserLexer:     parserLexer,
		reverseToken:    make([]lexer.Token, 0),
		registerNames:   []string{"t0", "t1", "t2", "t3", "t4", "t5", "t6", "t7"},
		regiserStack:    make([]string, 0),
		registerNameIdx: 0,
	}
}

func (a *AttributeParser) putbackToken(token lexer.Token) {
	a.reverseToken = append(a.reverseToken, token)
}

func (a *AttributeParser) getToken() lexer.Token {
	//先看看有没有上次退回去的 token
	if len(a.reverseToken) > 0 {
		token := a.reverseToken[len(a.reverseToken)-1]
		a.reverseToken = a.reverseToken[0 : len(a.reverseToken)-1]
		return token
	}

	token, err := a.parserLexer.Scan()
	if err != nil && token.Tag != lexer.EOF {
		sErr := fmt.Sprintf("get token with err:%s\n", err)
		panic(sErr)
	}

	return token
}

func (a *AttributeParser) match(tag lexer.Tag) bool {
	token := a.getToken()
	if token.Tag != tag {
		a.putbackToken(token)
		return false
	}

	return true
}

func (a *AttributeParser) newName() string {
	//返回一个寄存器的名字
	if a.registerNameIdx >= len(a.registerNames) {
		//没有寄存器可用
		panic("register name running out")
	}
	name := a.registerNames[a.registerNameIdx]
	a.registerNameIdx += 1
	return name
}

func (a *AttributeParser) freeName(name string) {
	//释放当前寄存器名字
	if a.registerNameIdx > len(a.registerNames) {
		panic("register name index out of bound")
	}

	if a.registerNameIdx == 0 {
		panic("register name is full")
	}

	a.registerNameIdx -= 1
	a.registerNames[a.registerNameIdx] = name
}

func (a *AttributeParser) Parse() {
	a.stmt()
}

func (a *AttributeParser) stmt() {
	for a.match(lexer.EOF) != true {
		t := a.newName()
		a.expr(t)
		a.freeName(t)
		if a.match(lexer.SEMI) != true {
			panic("missing ; at the end of expression")
		}
	}
}

func (a *AttributeParser) expr(t string) {
	a.term(t)
	a.expr_prime(t)
}

func (a *AttributeParser) expr_prime(t string) {
	if a.match(lexer.PLUS) {
		t2 := a.newName()
		a.term(t2)
		fmt.Printf("%s += %s\n", t, t2)
		a.freeName(t2)
		a.expr_prime(t)
	}
}

func (a *AttributeParser) term(t string) {
	a.factor(t)
	a.term_prime(t)
}

func (a *AttributeParser) term_prime(t string) {
	if a.match(lexer.MUL) {
		t2 := a.newName()
		a.factor(t2)
		fmt.Printf("%s *= %s\n", t, t2)
		a.freeName(t2)
		a.term_prime(t)
	}
}

func (a *AttributeParser) factor(t string) {
	if a.match(lexer.NUM) {
		fmt.Printf("%s = %s\n", t, a.parserLexer.Lexeme)
	} else if a.match(lexer.LEFT_BRACKET) {
		a.expr(t)
		if a.match(lexer.RIGHT_BRACKET) != true {
			panic("missing ) for expr")
		}
	}
}

```
我们可以看到 AttributeParser 跟我们前面实现的 AugmentedParser 区别不大，一个明显区别是，解析函数接受一个传进来的参数，这个参数可以看做是语法属性，他由语法表达式左边符号对应的函数创建然后传递给右边符号对应的函数。我们看如下代码：
```
func (a *AttributeParser) stmt() {
	for a.match(lexer.EOF) != true {
		t := a.newName()
		a.expr(t)
		a.freeName(t)
		if a.match(lexer.SEMI) != true {
			panic("missing ; at the end of expression")
		}
	}
}
```
stmt 函数在调用时创建了一个寄存器名称，然后调用 expr 时将该名称作为参数传入，在语法表达上相当于：
```
stmt_(t) -> expr_(t) SEMI stmt
```
其中 t 是左边 stmt 符号附带的参数，他将该参数传递给右边符号 expr，expr 利用该传过来的符号在语法解析时进行代码生成。从上面代码我们也能看出，它实际上是增强语法和属性语法的结合体，例如代码将属性作为参数传入，同时在解析的过程中又在特定位置执行特定步骤，因此上面的解析过程其实可以对应成如下的“增强属性语法”：
```go
stmt -> epsilon | {t=newName()} expt_(t) SEMI stmt
expr_(t) -> term_(t) expr_prime_(t)
expr_prime_(t) -> PLUS {t2 = newName()} term_(t2) {print(%s+=%s\n",t,t2) freenName(t2)} expr_prime_(t) | epsilon
term_(t) -> factor term_prime
term_prime_(t) -> MUL {t2 = newName()} factor_(t2) {print("%s+=%s\n",t,t2) freeName(t2)} term_prime_(t)
factor_(t) -> NUM {print("%s=*%s\n",t, lexeme)} | LEFT_PAREN expr_(t) RIGHT_PAREN
```
最后我们在 main.go 中调用属性语法解析器看看运行结果：
```
package main

import (
	"attribute_parser"
	"lexer"
)

func main() {
	exprLexer := lexer.NewLexer("1+2*(4+3);")
	attributeParser := attribute_parser.NewAttributeParser(exprLexer)
	attributeParser.Parse()
}

```
上面代码运行后结果如下：
```go
t0 = 1
t1 = 2
t2 = 4
t3 = 3
t2 += t3
t1 *= t2
t0 += t1
```
可以看到生成的结果跟我们上一节一样。更多内容请在 b 站搜索 coding 迪斯尼。
