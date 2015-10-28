#!/usr/bin/python2
import time
import sys
from random import random
import datetime
print """
S 1234 HOLE=`

S 4567 X=250 Y=250 BANK=0 MAP=4568 HOLE=`

S 999800 X=101 Y=101 BANK=0 MAP=99910 CELLWIDTH=7 BG=0 FG=0xFFFF00FF
S 9999 X=101 Y=101 BANK=0 MAP=99910 CELLWIDTH=7 BG=0x01000001
M 99910
                 _,--=--._
               ,'    _    `.
              -    _(_)_o   -
         ____'    /_  _/]    `____
  -=====::(+):::::::::::::::::(+)::=====-
           (+).""x""xx""x"",(+)
               .           ,
                 `  -=-  '

M 99911
+

P 0
"""

print "X 0 Test program starting."

sys.stdout.flush() #For the sake of java's too-stupid-to-live IO

sids = 0;

def getSid():
    global sids
    sids = sids+1
    return sids

class Thing:
    def __init__(self, x, y, r, dx, dy, dr, bank, map):
        self.sid = getSid()
        self.x = x
        self.y = y
        self.r = r
        self.dx = dx
        self.dy = dy
        self.dr = dr
        print "S "+str(self.sid)+" BG=0x01000001 BANK="+str(bank)+" MAP="+str(map)+"\n"

    def move(self):
        self.x = self.x+self.dx
        self.y = self.y+self.dy
        self.r = self.r+self.dr

        if self.x > 500:
            self.x = 0

        if self.y > 500:
            self.y = 0

        if self.x < 0:
            self.x = 500

        if self.y < 0:
            self.y = 500


    def draw(self):
        return "S " + str(self.sid) + " X="+str(self.x)+" Y="+str(self.y)+" ROT="+str(self.r)+"\n"

stars = []

for i in range(1000):
    stars.append(Thing(random()*500, random()*500, 0, random()-0.5, random()-0.5, (random()-0.5)*0.05, 0, 99911))

x=100;
y=100;
r=0;
s=1;
dx=0;
dy=0;
dr=0;
ds=0;

while True:
    while True:
        line = raw_input()
        #print "X 0 " + datetime.datetime.now().strftime('%Y/%m/%d %H:%M:%S.%f') + ":" + line
        parts = line.split()
        if len(parts)==0 or parts[0] == "P":
            break
        if len(parts)>2 and parts[0] == "KS":
            #print "X 0 "+line
            for part in parts[2:]:
                c = part[0]
                #print "X 0 ", c
                if c=='A':
                    x=x+5
                elif c=='D':
                    x=x-5
                elif c=='W':
                    y=y+5
                elif c=='S':
                    y=y-5
                elif c=='Q':
                    r=r-0.01
                elif c=='E':
                    r=r+0.01
                elif c=='R':
                    s=s-0.01
                elif c=='F':
                    s=s+0.01

    for star in stars:
       star.move()
       print star.draw()

    print "C 999800 Y="+str(y)+" X="+str(x)+" ROT="+str(r)+" SX="+str(s)+" SY="+str(s)+" VIEWWIDTH=512 VIEWHEIGHT=512 CELLWIDTH=4"
    print "P 16"
    sys.stdout.flush() #For the sake of java's too-stupid-to-live IO

  #  time.sleep(1.0/60.0)
