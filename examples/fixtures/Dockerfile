FROM scratch as A

ENV TEST=common-stage
ADD data/file-base /
ADD data/file-a /
CMD ["image-a"]

FROM scratch as B

ENV TEST=common-stage
ADD data/file-base /
ADD data/file-b /
CMD ["image-b"]

FROM scratch as AB

ENV TEST=common-stage
ADD data/file-base /
ADD data/file-a /
ADD data/file-b /
CMD ["image-ab"]
