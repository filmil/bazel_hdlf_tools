_TAGS = ["local", "requires-network"]

def lots_of_tags(name):
    native.genrule(
        name = name,
        tags = _TAGS,
    )
