import urwid
import os

class FileManager:
    def __init__(self, path):
        self.path = path
        self.contents = os.listdir(path)
        self.view = urwid.ListBox(urwid.SimpleFocusListWalker(self.get_items()))
        print(f"Initialized FileManager with path: {self.path}")

    def get_items(self):
        items = []
        for item in self.contents:
            if os.path.isdir(os.path.join(self.path, item)):
                item = f"[DIR] {item}"
            items.append(urwid.Text(item))
        return items

    def main(self):
        def handle_input(key):
            print(f"Key pressed: {key}")
            if key in ('q', 'Q'):
                raise urwid.ExitMainLoop()
            elif key == 'enter':
                focused_item = self.view.get_focus()[0]
                item_text = focused_item.get_text()[0]
                item_name = item_text[6:] if item_text.startswith("[DIR] ") else item_text
                new_path = os.path.join(self.path, item_name)
                if os.path.isdir(new_path):
                    print(f"Entering directory: {new_path}")
                    self.path = new_path
                    self.contents = os.listdir(new_path)
                    self.view.body = urwid.SimpleFocusListWalker(self.get_items())
                else:
                    print(f"Selected file: {new_path}")
            elif key == 'backspace':
                print("Going back to parent directory")
                self.path = os.path.dirname(self.path)
                self.contents = os.listdir(self.path)
                self.view.body = urwid.SimpleFocusListWalker(self.get_items())

        loop = urwid.MainLoop(self.view, unhandled_input=handle_input)
        loop.run()

if __name__ == "__main__":
    fm = FileManager(os.getcwd())
    fm.main()
