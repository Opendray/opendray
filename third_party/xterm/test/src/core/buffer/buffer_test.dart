import 'package:test/test.dart';
import 'package:xterm/xterm.dart';

void main() {
  group('Buffer.getText()', () {
    test('should return the text', () {
      final terminal = Terminal();
      terminal.write('Hello World');
      expect(terminal.buffer.getText(), startsWith('Hello World'));
    });

    test('can handle line wrap', () {
      final terminal = Terminal();
      terminal.resize(10, 10);

      final line1 = 'This is a long line that should wrap';
      final line2 = 'This is a short line';
      final line3 = 'This is a long long long long line that should wrap';
      final line4 = 'Short';

      terminal.write('$line1\r\n');
      terminal.write('$line2\r\n');
      terminal.write('$line3\r\n');
      terminal.write('$line4\r\n');

      final lines = terminal.buffer.getText().split('\n');
      expect(lines[0], line1);
      expect(lines[1], line2);
      expect(lines[2], line3);
      expect(lines[3], line4);
    });

    test('can handle negative start', () {
      final terminal = Terminal();

      terminal.write('Hello World');

      expect(
        terminal.buffer.getText(
          BufferRangeLine(CellOffset(-100, -100), CellOffset(100, 100)),
        ),
        startsWith('Hello World'),
      );
    });

    test('can handle invalid end', () {
      final terminal = Terminal();

      terminal.write('Hello World');

      expect(
        terminal.buffer.getText(
          BufferRangeLine(CellOffset(0, 0), CellOffset(100, 100)),
        ),
        startsWith('Hello World'),
      );
    });

    test('can handle reversed range', () {
      final terminal = Terminal();

      terminal.write('Hello World');

      expect(
        terminal.buffer.getText(
          BufferRangeLine(CellOffset(5, 5), CellOffset(0, 0)),
        ),
        startsWith('Hello World'),
      );
    });

    test('can handle block range', () {
      final terminal = Terminal();

      terminal.write('Hello World\r\n');
      terminal.write('Nice to meet you\r\n');

      expect(
        terminal.buffer.getText(
          BufferRangeBlock(CellOffset(2, 0), CellOffset(5, 1)),
        ),
        startsWith('llo\nce '),
      );
    });
  });

  group('Buffer.resize()', () {
    test('should resize the buffer', () {
      final terminal = Terminal();
      terminal.resize(10, 10);

      expect(terminal.viewWidth, 10);
      expect(terminal.viewHeight, 10);

      for (var i = 0; i < terminal.lines.length; i++) {
        final line = terminal.lines[i];
        expect(line.length, 10);
      }

      terminal.resize(20, 20);

      expect(terminal.viewWidth, 20);
      expect(terminal.viewHeight, 20);

      for (var i = 0; i < terminal.lines.length; i++) {
        final line = terminal.lines[i];
        expect(line.length, 20);
      }
    });
  });

  group('Buffer.deleteLines()', () {
    test('works', () {
      final terminal = Terminal();
      terminal.resize(10, 10);

      for (var i = 1; i <= 10; i++) {
        terminal.write('line$i');

        if (i < 10) {
          terminal.write('\r\n');
        }
      }

      terminal.setMargins(3, 7);
      terminal.setCursor(0, 5);

      terminal.buffer.deleteLines(1);

      expect(terminal.buffer.lines[2].toString(), 'line3');
      expect(terminal.buffer.lines[3].toString(), 'line4');
      expect(terminal.buffer.lines[4].toString(), 'line5');
      expect(terminal.buffer.lines[5].toString(), 'line7');
      expect(terminal.buffer.lines[6].toString(), 'line8');
      expect(terminal.buffer.lines[7].toString(), '');
      expect(terminal.buffer.lines[8].toString(), 'line9');
      expect(terminal.buffer.lines[9].toString(), 'line10');
    });
  });

  group('Buffer.insertLines()', () {
    test('works', () {
      final terminal = Terminal();

      for (var i = 0; i < 10; i++) {
        terminal.write('line$i\r\n');
      }

      print(terminal.buffer);

      terminal.setMargins(2, 6);
      terminal.setCursor(0, 4);

      print(terminal.buffer.absoluteCursorY);

      terminal.buffer.insertLines(1);

      print(terminal.buffer);

      expect(terminal.buffer.lines[3].toString(), 'line3');
      expect(terminal.buffer.lines[4].toString(), ''); // inserted
      expect(terminal.buffer.lines[5].toString(), 'line4'); // moved
      expect(terminal.buffer.lines[6].toString(), 'line5'); // moved
      expect(terminal.buffer.lines[7].toString(), 'line7');
    });

    test('has no effect if cursor is out of scroll region', () {
      final terminal = Terminal();

      for (var i = 0; i < 10; i++) {
        terminal.write('line$i\r\n');
      }

      terminal.setMargins(2, 6);
      terminal.setCursor(0, 1);

      terminal.buffer.insertLines(1);

      expect(terminal.buffer.lines[2].toString(), 'line2');
      expect(terminal.buffer.lines[3].toString(), 'line3');
      expect(terminal.buffer.lines[4].toString(), 'line4');
      expect(terminal.buffer.lines[5].toString(), 'line5');
      expect(terminal.buffer.lines[6].toString(), 'line6');
      expect(terminal.buffer.lines[7].toString(), 'line7');
    });
  });

  group('Buffer.getWordBoundary supports custom word separators', () {
    test('can set word separators', () {
      final terminal = Terminal(wordSeparators: {'o'.codeUnitAt(0)});

      terminal.write('Hello World');

      expect(
        terminal.mainBuffer.getWordBoundary(CellOffset(0, 0)),
        BufferRangeLine(CellOffset(0, 0), CellOffset(4, 0)),
      );

      expect(
        terminal.mainBuffer.getWordBoundary(CellOffset(5, 0)),
        BufferRangeLine(CellOffset(5, 0), CellOffset(7, 0)),
      );
    });
  });

  test('does not delete lines beyond the scroll region', () {
    final terminal = Terminal();
    terminal.resize(10, 10);

    for (var i = 1; i <= 10; i++) {
      terminal.write('line$i');

      if (i < 10) {
        terminal.write('\r\n');
      }
    }

    terminal.setMargins(3, 7);
    terminal.setCursor(0, 5);

    terminal.buffer.deleteLines(20);

    expect(terminal.buffer.lines[2].toString(), 'line3');
    expect(terminal.buffer.lines[3].toString(), 'line4');
    expect(terminal.buffer.lines[4].toString(), 'line5');
    expect(terminal.buffer.lines[5].toString(), '');
    expect(terminal.buffer.lines[6].toString(), '');
    expect(terminal.buffer.lines[7].toString(), '');
    expect(terminal.buffer.lines[8].toString(), 'line9');
    expect(terminal.buffer.lines[9].toString(), 'line10');
  });

  group('Buffer.eraseDisplayFromCursor()', () {
    test('works', () {
      final terminal = Terminal();
      terminal.resize(3, 3);
      terminal.write('123\r\n456\r\n789');

      terminal.setCursor(1, 1);
      terminal.buffer.eraseDisplayFromCursor();

      expect(terminal.buffer.lines[0].toString(), '123');
      expect(terminal.buffer.lines[1].toString(), '4');
      expect(terminal.buffer.lines[2].toString(), '');
    });
  });

  // Smoke tests around the codex crash path. The strict
  // regression is `IndexAwareCircularBuffer zero-copy ref shift
  // via []= leaves no dangling refs before insert` in
  // circular_buffer_test.dart — these two only exercise the
  // surrounding Buffer plumbing so we'd catch a future Buffer-
  // level change that re-introduces a different shape of the
  // same dangling-reference bug.
  group('Buffer scroll + insert smoke (codex crash neighbourhood)', () {
    test('scrollUp followed by insert past the region survives', () {
      // Direct reproduction of the codex crash without relying on
      // ANSI escape decoding: call scrollUp inside a margin (which
      // is what Buffer.index() does on IND/LF at marginBottom),
      // then call lines.insert past the region (which is what
      // Buffer.index() does in the marginTop==0 branch). Before
      // the fix, scrollUp leaves dangling line refs in the
      // backing array; insert's _moveChild walks one and trips
      // IndexedItem._move's `assert(attached)`.
      // Stay at the default 24-row viewport: maxLines must be
      // >= viewHeight (Buffer's ctor seeds viewHeight empty
      // lines), and Terminal.resize(_, h) pops from the same
      // ring, so any maxLines below the default viewHeight
      // trips an unrelated assertion inside resize.
      const viewHeight = 24;
      final terminal = Terminal(maxLines: viewHeight);
      final buffer = terminal.buffer;

      // Fill the ring so its _startIndex moves off zero — the
      // cyclic-index arithmetic is where the dangling reference
      // becomes observable.
      for (var i = 0; i < viewHeight * 3; i++) {
        terminal.write('row $i\r\n');
      }

      // Install a DECSTBM-style region [0..3] and scroll inside it.
      buffer.setVerticalMargins(0, 3);
      buffer.scrollUp(1);

      // Now ask insert() to walk through the dangling slots.
      // absoluteMarginBottom + 1 lands in the middle of the
      // backing array, which routes through the for-loop in
      // IndexAwareCircularBuffer.insert.
      buffer.lines.insert(
        buffer.absoluteMarginBottom + 1,
        buffer.lines[0],
      );

      // No crash = pass. Verify every slot is still attached as a
      // belt-and-braces guard against silent corruption.
      for (var i = 0; i < buffer.lines.length; i++) {
        expect(buffer.lines[i], isNotNull,
            reason: 'lines[$i] should not be null after scrollUp+insert');
      }
    });

    test(
      'DECSTBM scrolling region + full scrollback + lineFeed survives',
      () {
        // Reproduces the mobile Codex crash:
        //
        //   'package:xterm/.../circular_buffer.dart': Failed
        //   assertion: line 312 pos 12: 'attached': is not true.
        //   #2  IndexedItem._move
        //   #3  IndexAwareCircularBuffer._moveChild
        //   #4  IndexAwareCircularBuffer.insert
        //   #5  Buffer.index
        //   #6  Buffer.lineFeed
        //
        // Path: codex sets a DECSTBM scroll region inside the
        // main buffer, fills the scrollback to maxLines, then
        // line-feeds at the bottom of the region. scrollUp's
        // zero-copy `lines[i] = lines[i+1]` used to leave
        // dangling line refs in the backing array, which the
        // subsequent `lines.insert(absoluteMarginBottom + 1, …)`
        // tripped over.
        //
        // The crash needs three things together:
        //   1. main buffer (not alt — see buffer.dart line 238)
        //   2. marginTop == 0 with marginBottom < viewHeight - 1
        //   3. buffer.lines._length == maxLines (so insert() walks
        //      the _moveChild loop instead of taking the push path)
        //
        // maxLines must be >= viewHeight (Buffer ctor pushes
        // viewHeight empties), so we pick the smallest legal
        // value that still lets us fill the ring quickly.
        const viewHeight = 24;
        final terminal = Terminal(maxLines: viewHeight);

        // Push more lines than maxLines so the ring's
        // _startIndex moves off zero — the cyclic-index math is
        // exactly where the dangling reference manifests.
        for (var i = 0; i < viewHeight * 2; i++) {
          terminal.write('line $i\r\n');
        }

        // DECSTBM: scroll region rows 1..4 (1-indexed wire
        // protocol → marginTop=0, marginBottom=3). Codex pins a
        // status panel near the top with a similar shape.
        terminal.write('\x1b[1;4r');

        // Park the cursor on the bottom row of the region and
        // pump line feeds. Each LF at marginBottom runs
        // Buffer.index() → lines.insert(absoluteMarginBottom + 1,
        // …); before the fix, the dangling reference left by a
        // previous IND tripped IndexedItem._move's
        // assert(attached).
        terminal.write('\x1b[4;1H');
        for (var i = 0; i < viewHeight * 3; i++) {
          terminal.write('row $i\n');
        }

        // Reach this point at all = no crash. Sanity-check that
        // every line in the buffer is still a valid (attached)
        // BufferLine.
        for (var i = 0; i < terminal.buffer.lines.length; i++) {
          expect(terminal.buffer.lines[i], isNotNull,
              reason: 'lines[$i] should not be null after scroll');
        }
      },
    );
  });
}
