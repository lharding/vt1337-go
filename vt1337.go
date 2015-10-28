package main

import (
    "fmt"
    "strings"
    "strconv"
    "os"
    "io"
    "bufio"
    "log"
    "os/exec"
    "unicode/utf8"
    "container/list"
    "time"
// and now for a real-word example of why github URLs in source files is a terrible idea:
// https://docs.google.com/document/d/1zORKEEFPsJ5AujtPbtQYQquvAopuXb3whWud1sA7nAE/edit
    "github.com/go-gl/glfw/v3.0/glfw"
    "github.com/go-gl-legacy/gl"
    mathgl "github.com/go-gl/mathgl/mgl32"
    "runtime"
)

func errorCallback(err glfw.ErrorCode, desc string) {
    fmt.Printf("%v: %v\n", err, desc)
}

///////
// Utils
////////

func chkErr(err error) {
    if err != nil {
        log.Fatal(err)
        panic(err)
    }
}

const DEBUG = false
func debug(format string, args... interface{}) {
    if(DEBUG) {
        fmt.Printf(format+"\n", args...)
    }
}

////////
// Sprites
////////

type Sprite struct {
    x, y, z float32
    sx, sy, rot float32

    smap, bank uint

    fgcolor, bgcolor uint32

    cellwidth, cellheight int32
    hole rune
}

var spriteFieldParsers parserMap = parserMap {
    "X": parseFloat,
    "Y": parseFloat,
    "Z": parseFloat,

    "SX": parseFloat,
    "SY": parseFloat,

    "ROT": parseFloat,

    "MAP": parseUint,
    "BANK": parseUint,

    "FG": parseFourByte,
    "BG": parseFourByte,

    "CELLWIDTH": parseInt,
    "CELLHEIGHT": parseInt,

    "HOLE": parseNoop,
}


func (this *Sprite) makeMutationThunk(args ArgMap) MutationThunk {
    args.parse(spriteFieldParsers)

    start := time.Now()

    return func() {
        debug("Thunk delay: %v\n", time.Now().Sub(start))
        x, ok := args["X"]; if ok { this.x = x.(float32) }
        y, ok := args["Y"]; if ok { this.y = y.(float32) }
        z, ok := args["Z"]; if ok { this.z = z.(float32) }

        sx, ok := args["SX"]; if ok { this.sx = sx.(float32) }
        sy, ok := args["SY"]; if ok { this.sy = sy.(float32) }

        rot, ok := args["ROT"]; if ok { this.rot = rot.(float32) }

        smap, ok := args["MAP"]; if ok { this.smap = smap.(uint) }
        bank, ok := args["BANK"]; if ok { this.bank = bank.(uint) }

        fgcolor, ok := args["FGCOLOR"]; if ok { this.fgcolor = fgcolor.(uint32) }
        bgcolor, ok := args["BGCOLOR"]; if ok { this.bgcolor = bgcolor.(uint32) }

        cellwidth, ok := args["CELLWIDTH"]; if ok { this.cellwidth = int32(cellwidth.(int)) }
        cellheight, ok := args["CELLHEIGHT"]; if ok { this.cellheight = int32(cellheight.(int)) }

        hole, ok := args["HOLE"]; if ok { this.hole, _ = utf8.DecodeRune([]byte(hole.(string))) }
    }
}

func makeSprite() interface{} {
    s := new(Sprite)
    s.sx = 1
    s.sy = 1
    s.cellwidth = 8
    s.cellheight = 8
    return s
}

var sprites map[uint]*Sprite = make(map[uint]*Sprite)

func findSprite(id uint) *Sprite {
    sprite := findEntity(id, makeSprite).(*Sprite)
    sprites[id] = sprite
    return sprite
}

////////
// Maps
////////

type SMap struct {
    data []string
//    texture gl.Texture
}

func (m *SMap) getWidth() int32 {
    max := 0
    for _, line := range m.data {
        l := len(line)
        if l > max { max = l }
    }

    return int32(max)
}

func (m *SMap) getHeight() int32 {
    return int32(len(m.data))
}

func makeSMap() interface{} { return new(SMap) }

func findSMap(id uint) *SMap {
    return findEntity(id, makeSMap).(*SMap)
}

///////////
// Camera
///////////

var camera *Sprite = makeSprite().(*Sprite)
var camWidth float32 = 512
var camHeight float32 = 512

/////////
// Entity Management
/////////


var entities map[uint]interface{} = make(map[uint]interface{})

func findEntity(id uint, makeEnt func() interface{}) interface{} {
    found, ok := entities[id]
    if !ok {
        found = makeEnt()
        entities[id] = found
    }

    return found
}

func deleteEntity(id uint) {
    delete(entities, id)
    delete(sprites, id)
}

////////
// Command Processing
////////

const (
        CMD_SPRITE = 'S'
        CMD_MAP = 'M'
        CMD_DELETE = 'D'
        CMD_SPRITE_BANK = 'B'
        CMD_PAGE_FLIP = 'P'
        CMD_UNKNOWN = 'X'
        CMD_CAMERA = 'C'
)

type MutationThunk func()

const MAX_QUEUE_DEPTH = 100000
var commandBus = make(chan func(), MAX_QUEUE_DEPTH)

type parserFunc func(f interface{}) (interface{}, error)
type parserMap map[string]parserFunc

func parseFloat(f interface{}) (interface{}, error) {
    num, err := strconv.ParseFloat(f.(string), 10)
    return float32(num), err
}

func parseUint(f interface{}) (interface{}, error) {
    num, err := strconv.ParseUint(f.(string), 10, 32)
    return uint(num), err
}

func parseInt(f interface{}) (interface{}, error) {
    num, err := strconv.ParseInt(f.(string), 10, 32)
    return int(num), err
}

func parseFourByte(f interface{}) (interface{}, error) {
    num, err := strconv.ParseInt(f.(string), 0, 32)
    return int32(num), err
}

func parseNoop(f interface{}) (interface{}, error) {
    return f, nil
}

func (m ArgMap) parse(parsers parserMap) {
    for key, val := range m {
        parser, ok := parsers[key]
        if !ok {
            debug("Unknown field while parsing fields:", key)
            continue
        }

        m[key], _ = parser(val)
    }
}

type ArgMap map[string]interface{}
type cmdHandler func(uint, ArgMap, *bufio.Reader)

var tickLen uint = 12345678
var ticks <-chan time.Time = time.Tick(time.Duration(10)*time.Millisecond)

var cmdHandlers map[byte]cmdHandler = map[byte]cmdHandler{
    CMD_MAP: func(id uint, argmap ArgMap, input *bufio.Reader) {
        line := "START"
        height := 0
        lines := list.New()
        lines.Init()
        var err error
        for len(line)>0 && line[0]!='\n' {
            line, err = input.ReadString('\n'); chkErr(err)
            fmt.Printf("  Chomped map line: %v", line)
            lines.PushBack(line)
            height++
        }

        smap := findSMap(id)
        smap.data = make([]string, height)

        y := 0
        for e := lines.Front(); e != nil && y < height; e = e.Next() {
            smap.data[y] = e.Value.(string)
            y++
        }

        //coming soon...
        //smap.texture = gl.GenTexture()
    },
    CMD_SPRITE: func(id uint, argmap ArgMap, input *bufio.Reader) {
        sprite := findSprite(id)
        thunk := sprite.makeMutationThunk(argmap)
        commandBus <- thunk
    },
    CMD_CAMERA: func(id uint, argmap ArgMap, input *bufio.Reader) {
        camera = findSprite(id)
        thunk := camera.makeMutationThunk(argmap)
        commandBus <- thunk
        newWidth, ok := argmap["VIEWWIDTH"]; if ok { ugh, _ := parseFloat(newWidth); camWidth = ugh.(float32) }
        newHeight, ok := argmap["VIEWHEIGHT"]; if ok { ugh, _ := parseFloat(newHeight); camHeight = ugh.(float32) }
    },
    CMD_PAGE_FLIP: func(delay uint, argmap ArgMap, input *bufio.Reader) {
        if delay == 0 { delay = 1 }
        if delay != tickLen {
            ticks = time.Tick(time.Duration(delay)*time.Millisecond)
            tickLen = delay
            fmt.Printf("New frame length: %v", delay)
        }
    },
    CMD_DELETE: func(id uint, argmap ArgMap, input *bufio.Reader) {
        commandBus <- func() { deleteEntity(id) }
    },
}

func parseCommand(tokens []string) (byte, uint, ArgMap) {
    argmap := make(ArgMap)
    command := tokens[0][0]
    id, err := strconv.ParseUint(tokens[1], 10, 64); chkErr(err)

    if len(tokens) > 2 {
        for _, token := range tokens[2:] {
            pair := strings.Split(token, "=")
            if len(pair) < 2 {
                debug("Warning, unreadable token %v", token)
                continue
            }

            argmap[pair[0]] = pair[1]
        }
    }

    return command, uint(id), argmap
}

var decodeTime = time.Duration(0)
var decodes = 0

func readCommands(input *bufio.Reader) {
    var line string
    var err error

    for len(line)==0 || line[0]!=CMD_PAGE_FLIP {
        line, err = input.ReadString('\n'); chkErr(err)

        //fmt.Printf("Processing line: %v", line)

        then := time.Now()

        tokens := strings.Split(line[0:len(line)-1], " ")
        if len(tokens) < 2 {
            continue
        }

        // Special case for logging from client app
        if tokens[0] == "X" {
            //log.Println(line)
            fmt.Printf("%v: %v", time.Now().Format(time.StampMilli), line)
            continue
        }

        command, id, argmap := parseCommand(tokens)

        handler, ok := cmdHandlers[command]
        if ok {
            handler(id, argmap, input)
        } else {
            debug("Unknown command %q\n", command)
        }

        decodeTime += time.Since(then)
        decodes++

        //fmt.Printf("got command %q, id=%v:\n", command, id)
        //for key, val := range argmap {
        //    fmt.Printf("   %v = %v\n", key, val)
        //}
    }

    debug("got P line, returning.")
}

//////////////////
// Input Handling
//////////////////
var keysDown map[glfw.Key]bool = make(map[glfw.Key]bool)

func handleKey(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
    keysDown[key] = action != glfw.Release
}

func sendInput(out io.Writer) {
    fmt.Fprintf(out, "KS 0 ")
    for key, state := range keysDown {
        if state {
            fmt.Fprintf(out, "%c%d ", key, 0)
            //fmt.Printf("%c%d ", key, 0)
        }
    }

    fmt.Fprintf(out, "\nP\n")
    //fmt.Printf("\nP\n")
    //out.Flush()
}

//////////////////
// OpenGL
//////////////////

const (
	vertex = `#version 330
in vec2 position;
uniform mat4 modelView;
uniform mat4 projection;
uniform vec2 size;
void main()
{
    gl_Position = projection*modelView*vec4(position*size, 0.0, 1.0);
}`

	fragment = `#version 330
out vec4 outColor;
void main()
{
    outColor = vec4(1.0, 1.0, 1.0, 0.5);
}`
)

//////////////////
// Main
//////////////////

func main() {
    glfw.SetErrorCallback(errorCallback)
    // lock glfw/gl calls to a single thread
    runtime.LockOSThread()
    runtime.GOMAXPROCS(8) // Render, read commands, send input, extra for file loading, etc

    if !glfw.Init() {
        panic("Could not init glfw!")
    }

    defer glfw.Terminate()

    glfw.WindowHint(glfw.ContextVersionMajor, 3)
    glfw.WindowHint(glfw.ContextVersionMinor, 3)
    glfw.WindowHint(glfw.OpenglForwardCompatible, glfw.True)
    glfw.WindowHint(glfw.OpenglProfile, glfw.OpenglCoreProfile)

    window, err := glfw.CreateWindow(800, 600, "Example", nil, nil)
    if err != nil {
            panic(err)
    }

    window.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
        fmt.Printf("Framebuffer size is now %vx%v\n", width, height)
        // Keep aspect ratio from camwidth/camheight
        camRatio := camWidth/camHeight
        bufRatio := float32(width)/float32(height)
        var newWidth, newHeight float32

        switch {
            case camRatio > bufRatio:
                newHeight = float32(width)/camRatio
                newWidth = float32(width)
            case bufRatio > camRatio:
                newWidth = float32(height)*camRatio
                newHeight = float32(height)
        }

        fmt.Printf("Viewport size is now %vx%v; cam ratio is %v; viewport ratio is %v;\n", newWidth, newHeight, camRatio, newWidth/newHeight)

        gl.Viewport((width-int(newWidth))/2, (height-int(newHeight))/2, int(newWidth), int(newHeight));
    })

    defer window.Destroy()

    window.MakeContextCurrent()
    glfw.SwapInterval(1)

    gl.Init()

    // Enable blending
    gl.Enable(gl.BLEND);
    gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);

    vao := gl.GenVertexArray()
    vao.Bind()

    vbo := gl.GenBuffer()
    vbo.Bind(gl.ARRAY_BUFFER)

    verticies := []float32{-0.5,  0.5,
                            0.5,  0.5,
                            0.5, -0.5,
                           -0.5, -0.5}

    gl.BufferData(gl.ARRAY_BUFFER, len(verticies)*4, verticies, gl.STATIC_DRAW)

    vertex_shader := gl.CreateShader(gl.VERTEX_SHADER)
    vertex_shader.Source(vertex)
    vertex_shader.Compile()
    fmt.Println(vertex_shader.GetInfoLog())
    defer vertex_shader.Delete()

    fragment_shader := gl.CreateShader(gl.FRAGMENT_SHADER)
    fragment_shader.Source(fragment)
    fragment_shader.Compile()
    fmt.Println(fragment_shader.GetInfoLog())
    defer fragment_shader.Delete()

    program := gl.CreateProgram()
    program.AttachShader(vertex_shader)
    program.AttachShader(fragment_shader)

    program.BindFragDataLocation(0, "outColor")
    program.Link()
    program.Use()
    defer program.Delete()

    positionAttrib := program.GetAttribLocation("position")
    positionAttrib.AttribPointer(2, gl.FLOAT, false, 0, nil)
    positionAttrib.EnableArray()
    defer positionAttrib.DisableArray()

    modelMat := program.GetUniformLocation("modelView")
    projMat := program.GetUniformLocation("projection")
    spriteSize := program.GetUniformLocation("size")

    cmd := exec.Command(os.Args[1], os.Args[2:]...)
    stdoutReader, stdoutWriter := io.Pipe()
    cmd.Stdout = stdoutWriter
    input := bufio.NewReader(stdoutReader)

    stdinReader, stdinWriter := io.Pipe()
    cmd.Stdin = stdinReader

    stderr, err := cmd.StderrPipe(); chkErr(err)

    go io.Copy(os.Stderr, stderr)

    err = cmd.Start(); chkErr(err)

    window.SetKeyCallback(handleKey)

    go func() {
        runtime.LockOSThread()
        for !window.ShouldClose() {
            sendInput(stdinWriter)
            //fmt.Fprintf(stdinWriter, "T %v\n", time.Now())
            <-ticks
            readCommands(input)
        }
    }()


    frameCnt := 0
    queueDepth := 0
    then := time.Now()

    for !window.ShouldClose() {
        frameCnt++
        queueDepth += len(commandBus)
        if frameCnt % 120 == 0 {
            fmt.Printf("Queue depth: %v. Render time: %v. Decode time: %v.\n", queueDepth/120, time.Since(then)/120, decodeTime/time.Duration(decodes))
            queueDepth = 0
            then = time.Now()
            decodeTime = time.Duration(0)
            decodes = 0
        }

        for len(commandBus)>0 {
            (<-commandBus)()
        }

        halfwidth := camWidth/2.0
        halfheight := camHeight/2.0
        projection := mathgl.Ortho2D(camera.x-halfwidth, camera.x+halfwidth, camera.y+halfheight, camera.y-halfheight)
        projection = mathgl.Scale3D(camera.sx, camera.sy, 1).Mul4(projection)
        projection = mathgl.HomogRotate3DZ(camera.rot).Mul4(projection)

        projMat.UniformMatrix4f(false, (*[16]float32)(&projection))

        gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

        for _, sprite := range sprites {
            sMat := mathgl.Scale3D(sprite.sx, sprite.sy, 1)
            sMat = mathgl.HomogRotate3DZ(sprite.rot).Mul4(sMat)
            sMat = mathgl.Translate3D(sprite.x, sprite.y, sprite.z).Mul4(sMat)

            // temp hack bank in
            sprite.bank = 1

            sMap := findSMap(sprite.smap)

            spriteSize.Uniform2f(float32(sMap.getWidth()*sprite.cellwidth), float32(sMap.getHeight()*sprite.cellheight))

            modelMat.UniformMatrix4f(false, (*[16]float32)(&sMat))
            gl.DrawArrays(gl.TRIANGLE_FAN, 0, 4)
        }

        window.SwapBuffers()
        glfw.PollEvents()

        if window.GetKey(glfw.KeyEscape) == glfw.Press {
                window.SetShouldClose(true)
        }
    }
}
