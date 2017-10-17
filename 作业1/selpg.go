package main

import (
  "fmt"
  "os"
  "flag"
  "io"
  "os/exec"
  "bufio"
  "io/ioutil"
)

type selpg_args struct {
  start_page int
  end_page int
  page_len int
  page_type bool
  print_dest string
  input_file string
}

//process and check the args from the command line
func process_args() *selpg_args {
  start := flag.Int("s", -0, "the number of start_page(from 1)")
  end := flag.Int("e", -0, "the number of the end_page(from 1, must bigger than start)")
  page_type := flag.Bool("f", true, "true means seperate the page by feed, flase and default means sperate the page by the lines per page")
  page_len := flag.Int("l", 72, "the number of lines for each page(default is 72 per page, the length of per page mush bigger than 1 but less than 1 >> 31 - 1)")
  dest := flag.String("d", "", "the destination of printer, default will output to the stdout")

  flag.Parse()

  var parsed_arg selpg_args
  // fmt.Printf("start: %+v, end: %+v, page_type: %+v, page_len: %+v, dest: %+v\n",
  // *start, *end, *page_type, *page_len, *dest)
  if *start < 1 || *start > (1 << 31 - 1) {
    flag.Usage()
    os.Exit(1)
  }
  parsed_arg.start_page = *start

  if *end < 1 || *end > (1 << 31 - 1) || *end < *start {
    flag.Usage()
    os.Exit(2)
  }
  parsed_arg.end_page = *end

  //because the value of -f mush be true or false, it is not necessary to check
  parsed_arg.page_type = *page_type

  if *page_len < 1 || *page_len > (1 << 31 - 1) {
    flag.Usage()
    os.Exit(3)
  }
  parsed_arg.page_len = *page_len

  parsed_arg.print_dest = *dest

  //get the file
  not_flag_args := flag.Args()
  if len(not_flag_args) == 1 {
    _, err := os.Stat(not_flag_args[0])
    if err != nil {
      if os.IsNotExist(err) {
        fmt.Printf("Error: \n", err)
        os.Exit(4)
      }
    }

    file, err2 := os.OpenFile(not_flag_args[0], os.O_RDONLY, 0666)
    if err2 != nil {
      if os.IsPermission(err2) {
        fmt.Printf("input file \"%v\" exists but cannot be read", not_flag_args[0])
        os.Exit(5)
      }
    }
    file.Close()
    parsed_arg.input_file = not_flag_args[0]
    // fmt.Println(not_flag_args)
  } else {
    parsed_arg.input_file = ""
  }

  return &parsed_arg
}

func process_input(parsed_args *selpg_args) {
  var fin *os.File
  var fout *os.File
  var page_ctr int
  var line_ctr int
  if parsed_args.input_file == "" {
    fin = os.Stdin
  } else {
    temp_fin, err := os.Open(parsed_args.input_file)
    if err != nil {
      fmt.Fprintf(os.Stderr, "%v: could not open input file \"%v\"\n", os.Args[0], parsed_args.input_file)
      os.Exit(6)
    } else {
      fin = temp_fin
    }
  }
  readFile := bufio.NewReader(fin)

  var cmd *exec.Cmd
  var cmdin io.WriteCloser
  var cmdout io.ReadCloser
  var cmderr io.ReadCloser
  if parsed_args.print_dest == "" {
    fout = os.Stdout
  } else {
    cmd = exec.Command("lp", "-d", parsed_args.print_dest)
    temp_cmdin, err1 := cmd.StdinPipe()
    temp_cmdout, err2 := cmd.StdoutPipe()
    temp_cmderr, err3 := cmd.StderrPipe()
    if err1 != nil || err2 != nil || err3 != nil {
      fmt.Fprintf(os.Stderr, "%v: could not open pipe to \"%v\"\n", os.Args[0], parsed_args.print_dest)
      os.Exit(7)
    }
    cmdin = temp_cmdin
    cmdout = temp_cmdout
    cmderr = temp_cmderr
    //finish later
    cmd.Start()
  }

  if parsed_args.page_type == false {
    line_ctr = 0
    page_ctr = 1

    for true {
      line, err := readFile.ReadString('\n')
      if err == io.EOF {
        break
      }
      line_ctr++
      if line_ctr > parsed_args.page_len {
        page_ctr++
        line_ctr = 1
      }
      if page_ctr >= parsed_args.start_page && page_ctr <= parsed_args.end_page {
        if fout != nil {
          fout.WriteString(line)  //参数是字节数组
        } else {
          temp := []byte(line)
          fmt.Println(line)
          cmdin.Write(temp)
          cmdin.Write([]byte{'\n'})
        }
      }
    }
  } else {
    page_ctr = 1
    readFile := bufio.NewReader(fin)
    for true {
      char, err := readFile.ReadByte()
      if err == io.EOF {
        break
      }
      if char == '\f' {
        page_ctr++
      }
      if page_ctr >= parsed_args.start_page && page_ctr <= parsed_args.end_page {
        if fout != nil {
          fout.Write([]byte{char})
        } else {
          cmdin.Write([]byte{char})
        }
      }
    }
  }

  if cmd != nil {
    cmdin.Close()

    var temp []byte
    temp, _ = ioutil.ReadAll(cmdout)
    cmdout.Close()
    if len(temp) != 0 {
      fmt.Println(string(temp))
    }

    temp, _ = ioutil.ReadAll(cmderr)
    cmderr.Close()
    if len(temp) != 0 {
      fmt.Println(string(temp))
    }

    cmd.Wait()
  }

  //使用os.Stderr
  if page_ctr < parsed_args.start_page {
    fmt.Fprintf(os.Stderr, "start_page (%v) greater than total pages (%v), no output written\n", parsed_args.start_page, page_ctr)
  } else if page_ctr < parsed_args.end_page {
    fmt.Fprintf(os.Stderr, "end_page (%v) greater than total pages (%v) less output than expected\n", parsed_args.end_page, page_ctr)
  }

  fin.Close()
  fout.Close()
}

func main()  {
  parsed_args := process_args()
  process_input(parsed_args)
}
