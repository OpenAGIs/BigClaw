import argparse

from .service import run_server


def main() -> None:
    parser = argparse.ArgumentParser(prog="bigclaw", description="BigClaw developer utilities")
    sub = parser.add_subparsers(dest="command")

    serve = sub.add_parser("serve", help="Run local BigClaw static web server")
    serve.add_argument("--host", default="127.0.0.1")
    serve.add_argument("--port", type=int, default=8008)
    serve.add_argument("--dir", default="reports")

    args = parser.parse_args()

    if args.command == "serve":
        run_server(host=args.host, port=args.port, directory=args.dir)
        return

    parser.print_help()


if __name__ == "__main__":
    main()
