from tkinter import Y

from itsdangerous import json

data = {
    "SCREEN": {
        "SIZE": [
            3120,
            1440
        ]
    },
    "MOUSE": {
        "SWITCH_KEY": "KEY_GRAVE",
        "POS": [
            1660,
            720
        ],
        "SPEED": [
            1,
            1
        ]
    },
    "WHEEL": {
        "POS": [
            400,
            1040
        ],
        "RANGE": 300,
        "WASD": [
            "KEY_W",
            "KEY_A",
            "KEY_S",
            "KEY_D"
        ]
    },
    "KEY_MAPS": {
        "BTN_RT_2": {
            "TYPE": "PRESS",
            "POS": [
                711,
                154
            ]
        },
        "BTN_LT_2": {
            "TYPE": "PRESS",
            "POS": [
                381,
                2450
            ]
        },
        "BTN_X": {
            "TYPE": "PRESS",
            "POS": [
                116,
                2496
            ]
        },
        "BTN_LB": {
            "TYPE": "PRESS",
            "POS": [
                461,
                3004
            ]
        },
        "BTN_RB": {
            "TYPE": "PRESS",
            "POS": [
                109,
                2761
            ]
        },
        "BTN_DPAD_DOWN": {
            "TYPE": "PRESS",
            "POS": [
                261,
                1552
            ]
        },
        "BTN_DPAD_LEFT": {
            "TYPE": "PRESS",
            "POS": [
                1276,
                2816
            ]
        },
        "BTN_DPAD_UP": {
            "TYPE": "PRESS",
            "POS": [
                1118,
                2816
            ]
        },
        "BTN_DPAD_RIGHT": {
            "TYPE": "PRESS",
            "POS": [
                974,
                2818
            ]
        },
        "BTN_A": {
            "TYPE": "PRESS",
            "POS": [
                432,
                3022
            ]
        },
        "BTN_B": {
            "TYPE": "PRESS",
            "POS": [
                116,
                2730
            ]
        },
        "BTN_LS": {
            "TYPE": "PRESS",
            "POS": [
                213,
                3009
            ]
        },
        "BTN_RS": {
            "TYPE": "PRESS",
            "POS": [
                121,
                2475
            ]
        },
        "BTN_SELECT": {
            "TYPE": "PRESS",
            "POS": [
                1376,
                1523
            ]
        },
        "KEY_Q": {
            "TYPE": "PRESS",
            "POS": [
                259,
                1552
            ]
        },
        "KEY_1": {
            "TYPE": "PRESS",
            "POS": [
                1245,
                2857
            ]
        },
        "KEY_2": {
            "TYPE": "PRESS",
            "POS": [
                1120,
                2850
            ]
        },
        "KEY_3": {
            "TYPE": "PRESS",
            "POS": [
                963,
                2845
            ]
        },
        "BTN_LEFT": {
            "TYPE": "PRESS",
            "POS": [
                724,
                154
            ]
        },
        "BTN_RIGHT": {
            "TYPE": "PRESS",
            "POS": [
                366,
                2457
            ]
        },
        "KEY_E": {
            "TYPE": "PRESS",
            "POS": [
                373,
                2485
            ]
        },
        "KEY_R": {
            "TYPE": "PRESS",
            "POS": [
                120,
                2491
            ]
        },
        "KEY_SPACE": {
            "TYPE": "PRESS",
            "POS": [
                447,
                3004
            ]
        },
        "KEY_LEFTSHIFT": {
            "TYPE": "PRESS",
            "POS": [
                113,
                2759
            ]
        },
        "KEY_C": {
            "TYPE": "PRESS",
            "POS": [
                218,
                3011
            ]
        },
        "KEY_TAB": {
            "TYPE": "PRESS",
            "POS": [
                1351,
                1556
            ]
        }
    }
}




for btn in data["KEY_MAPS"]:
    ox,oy = data["KEY_MAPS"][btn]["POS"]
    nx = oy
    ny = 1440 - ox
    data["KEY_MAPS"][btn]["POS"] = [nx,ny]

print(json.dumps(data,indent=4))
